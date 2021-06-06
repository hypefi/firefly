// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"context"
	"encoding/json"
	"io"

	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

// SequencedBroadcastBatch is called in-line with a particular ledger's stream of events, so while we
// block here this blockchain event remains un-acknowledged, and no further events will arrive from this
// particular ledger.
//
// We must block here long enough to get the payload from the publicstorage, persist the messages in the correct
// sequence, and also persist all the data.
func (em *eventManager) BatchPinComplete(bi blockchain.Plugin, batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {

	log.L(em.ctx).Infof("-> SequencedBroadcastBatch txn=%s author=%s", protocolTxID, signingIdentity)
	defer func() {
		log.L(em.ctx).Infof("<- SequencedBroadcastBatch txn=%s author=%s", protocolTxID, signingIdentity)
	}()
	log.L(em.ctx).Tracef("SequencedBroadcastBatch info: %+v", additionalInfo)

	if batchPin.BatchPaylodRef != nil {
		return em.handleBroadcastPinComplete(batchPin, signingIdentity, protocolTxID, additionalInfo)
	}
	return em.handlePrivatePinComplete(batchPin, signingIdentity, protocolTxID, additionalInfo)
}

func (em *eventManager) handlePrivatePinComplete(batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	// Here we simple record all the pins as parked, and emit an event for the aggregator
	// to check whether the messages in the batch have been written.
	return em.retry.Do(em.ctx, "persist pins", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			err := em.persistBatchTransaction(ctx, batchPin, signingIdentity, protocolTxID, additionalInfo)
			if err == nil {
				err = em.persistPins(ctx, batchPin)
				if err == nil {
					err = em.emitPinnedEvent(ctx, batchPin)
				}
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}

func (em *eventManager) persistBatchTransaction(ctx context.Context, batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	// Get any existing record for the batch transaction record
	tx, err := em.database.GetTransactionByID(ctx, batchPin.TransactionID)
	if err != nil {
		return err // a peristence failure here is considered retryable (so returned)
	}
	if err := fftypes.ValidateFFNameField(ctx, batchPin.Namespace, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid batch '%s'. Transaction '%s' invalid namespace '%s': %a", batchPin.BatchID, batchPin.TransactionID, batchPin.Namespace, err)
		return nil // This is not retryable. skip this batch
	}
	if tx == nil {
		// We're the first to write the transaction record on this node
		tx = &fftypes.Transaction{
			ID: batchPin.TransactionID,
			Subject: fftypes.TransactionSubject{
				Namespace: batchPin.Namespace,
				Type:      fftypes.TransactionTypeBatchPin,
				Signer:    signingIdentity,
				Reference: batchPin.TransactionID,
			},
		}
		tx.Hash = tx.Subject.Hash()
	} else if tx.Subject.Type != fftypes.TransactionTypeBatchPin ||
		tx.Subject.Signer != signingIdentity ||
		tx.Subject.Reference == nil ||
		*tx.Subject.Reference != *batchPin.BatchID ||
		tx.Subject.Namespace != batchPin.Namespace {
		log.L(ctx).Errorf("Invalid batch '%s'. Existing transaction '%s' does not match batch subject", batchPin.BatchID, tx.ID)
		return nil // This is not retryable. skip this batch
	}

	// Set the updates on the transaction
	tx.ProtocolID = protocolTxID
	tx.Info = additionalInfo
	tx.Status = fftypes.OpStatusSucceeded

	// Upsert the transaction, ensuring the hash does not change
	err = em.database.UpsertTransaction(ctx, tx, true, false)
	if err != nil {
		if err == database.HashMismatch {
			log.L(ctx).Errorf("Invalid batch '%s'. Transaction '%s' hash mismatch with existing record", batchPin.BatchID, tx.Hash)
			return nil // This is not retryable. skip this batch
		}
		log.L(ctx).Errorf("Failed to insert transaction for batch '%s': %s", batchPin.BatchID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}

func (em *eventManager) persistCon(ctx context.Context, batchPin *blockchain.BatchPin) error {
	for _, pin := range batchPin.Pins {
		if err := em.database.InsertParked(ctx, &fftypes.Parked{
			Pin:     pin,
			Batch:   batchPin.BatchID,
			Created: fftypes.Now(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (em *eventManager) emitPinnedEvent(ctx context.Context, batchPin *blockchain.BatchPin) error {
	// Persist a batch pinned even
	event := fftypes.NewEvent(fftypes.EventTypesBatchPinned, batchPin.Namespace, batchPin.BatchID)
	if err := em.database.UpsertEvent(ctx, event, false); err != nil {
		log.L(ctx).Errorf("Failed to insert %s event for batch '%s': %s", event.Type, batchPin.BatchID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}
	return nil
}

func (em *eventManager) handleBroadcastPinComplete(batchPin *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	var body io.ReadCloser
	if err := em.retry.Do(em.ctx, "retrieve data", func(attempt int) (retry bool, err error) {
		body, err = em.publicstorage.RetrieveData(em.ctx, batchPin.BatchPaylodRef)
		return err != nil, err // retry indefinitely (until context closes)
	}); err != nil {
		return err
	}
	defer body.Close()

	var batch *fftypes.Batch
	err := json.NewDecoder(body).Decode(&batch)
	if err != nil {
		log.L(em.ctx).Errorf("Failed to parse payload referred in batch ID '%s' from transaction '%s'", batchPin.BatchID, protocolTxID)
		return nil // log and swallow unprocessable data
	}
	body.Close()

	// At this point the batch is parsed, so any errors in processing need to be considered as:
	// 1) Retryable - any transient error returned by processBatch is retried indefinitely
	// 2) Swallowable - the data is invalid, and we have to move onto subsequent messages
	// 3) Server shutting down - the context is cancelled (handled by retry)
	return em.retry.Do(em.ctx, "persist batch", func(attempt int) (bool, error) {
		// We process the batch into the DB as a single transaction (if transactions are supported), both for
		// efficiency and to minimize the chance of duplicates (although at-least-once delivery is the core model)
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			err := em.persistBatchTransaction(ctx, batchPin, signingIdentity, protocolTxID, additionalInfo)
			if err == nil {
				err = em.persistBatch(ctx, batch, signingIdentity, protocolTxID, additionalInfo)
				if err == nil {
					err = em.emitPinnedEvent(ctx, batchPin)
				}
				return err
			}
			return err
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}

// persistBatch performs very simple validation on each message/data element (hashes) and either persists
// or discards them. Errors are returned only in the case of database failures, which should be retried.
func (em *eventManager) persistBatch(ctx context.Context /* db TX context*/, batch *fftypes.Batch, author string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	l := log.L(ctx)
	now := fftypes.Now()

	if batch.ID == nil || batch.Payload.TX.ID == nil {
		l.Errorf("Invalid batch '%s'. Missing ID (%v) or payload ID (%v)", batch.ID, batch.ID, batch.Payload.TX.ID)
		return nil // This is not retryable. skip this batch
	}

	// Verify the hash calculation
	hash := batch.Payload.Hash()
	if batch.Hash == nil || *batch.Hash != *hash {
		l.Errorf("Invalid batch '%s'. Hash does not match payload. Found=%s Expected=%s", batch.ID, hash, batch.Hash)
		return nil // This is not retryable. skip this batch
	}

	// Verify the author matches
	id, err := em.identity.Resolve(ctx, batch.Author)
	if err != nil {
		l.Errorf("Invalid batch '%s'. Author '%s' cound not be resolved: %s", batch.ID, batch.Author, err)
		return nil // This is not retryable. skip this batch
	}
	if author != id.OnChain {
		l.Errorf("Invalid batch '%s'. Author '%s' does not match transaction submitter '%s'", batch.ID, id.OnChain, author)
		return nil // This is not retryable. skip this batch
	}

	// Set confirmed on the batch (the messages should not be confirmed at this point - that's the aggregator's job)
	batch.Confirmed = now

	// Upsert the batch itself, ensuring the hash does not change
	err = em.database.UpsertBatch(ctx, batch, true, false)
	if err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid batch '%s'. Batch hash mismatch with existing record", batch.ID)
			return nil // This is not retryable. skip this batch
		}
		l.Errorf("Failed to insert batch '%s': %s", batch.ID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	// Insert the data entries
	for i, data := range batch.Payload.Data {
		if err = em.persistBatchData(ctx, batch, i, data); err != nil {
			return err
		}
	}

	// Insert the message entries
	for i, msg := range batch.Payload.Messages {
		if err = em.persistBatchMessage(ctx, batch, i, msg); err != nil {
			return err
		}
	}

	return nil

}

func (em *eventManager) persistBatchData(ctx context.Context /* db TX context*/, batch *fftypes.Batch, i int, data *fftypes.Data) error {
	l := log.L(ctx)
	l.Tracef("Batch %s data %d: %+v", batch.ID, i, data)

	if data == nil {
		l.Errorf("null data entry %d in batch '%s'", i, batch.ID)
		return nil // skip data entry
	}

	hash := data.Value.Hash()
	if data.Hash == nil || *data.Hash != *hash {
		l.Errorf("Invalid data entry %d in batch '%s'. Hash does not match value. Found=%s Expected=%s", i, batch.ID, hash, data.Hash)
		return nil // skip data entry
	}

	// Insert the data, ensuring the hash doesn't change
	if err := em.database.UpsertData(ctx, data, true, false); err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid data entry %d in batch '%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, data.ID, data.Hash)
			return nil // This is not retryable. skip this data entry
		}
		l.Errorf("Failed to insert data entry %d in batch '%s': %s", i, batch.ID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}

func (em *eventManager) persistBatchMessage(ctx context.Context /* db TX context*/, batch *fftypes.Batch, i int, msg *fftypes.Message) error {
	l := log.L(ctx)
	l.Tracef("Batch %s message %d: %+v", batch.ID, i, msg)

	if msg == nil {
		l.Errorf("null message entry %d in batch '%s'", i, batch.ID)
		return nil // skip entry
	}

	if msg.Header.Author != batch.Author {
		l.Errorf("Mismatched author '%s' on message entry %d in batch '%s'", msg.Header.Author, i, batch.ID)
		return nil // skip entry
	}

	err := msg.Verify(ctx)
	if err != nil {
		l.Errorf("Invalid message entry %d in batch '%s': %s", i, batch.ID, err)
		return nil // skip message entry
	}

	// Insert the message, ensuring the hash doesn't change.
	// We do not mark it as confirmed at this point, that's the job of the aggregator.
	if err = em.database.UpsertMessage(ctx, msg, true, false); err != nil {
		if err == database.HashMismatch {
			l.Errorf("Invalid message entry %d in batch '%s'. Hash mismatch with existing record with same UUID '%s' Hash=%s", i, batch.ID, msg.Header.ID, msg.Hash)
			return nil // This is not retryable. skip this data entry
		}
		l.Errorf("Failed to insert message entry %d in batch '%s': %s", i, batch.ID, err)
		return err // a peristence failure here is considered retryable (so returned)
	}

	return nil
}
