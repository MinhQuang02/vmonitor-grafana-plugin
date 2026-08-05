import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { VMonitorQuery, VMonitorDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, VMonitorQuery, VMonitorDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
