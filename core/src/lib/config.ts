import { promisify } from 'util';
import { readFile } from 'fs';
import Ajv from 'ajv';
import configSchema from '../schemas/config.json';
import * as utils from './utils';
import { IConfig } from './interfaces';
import path from 'path';

const asyncReadFile = promisify(readFile);
const ajv = new Ajv();
const validateConfig = ajv.compile(configSchema);

export let config: IConfig;

export const initConfig = async () => {
  try {
    const data = JSON.parse(await asyncReadFile(path.join(utils.constants.DATA_DIRECTORY, utils.constants.CONFIG_FILE_NAME), 'utf8'));
    if(validateConfig(data)) {
      config = data;
    } else {
      throw new Error('Invalid configuration file');
    }
  } catch(err) {
    throw new Error(`Failed to read configuration file. ${err}`);
  }
};