import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export type StatisticType = 'avg' | 'min' | 'max' | 'sum' | 'count' | 'rate_1s' | 'rate_1m' | 'rate_5m';
export type RankType = 'topk' | 'bottomk';
export type RollupType = 'avg' | 'min' | 'max' | 'sum' | 'count';
export type RateType = 'per_second' | 'per_minute' | 'per_hour';
export type ReductionType = 'avg' | 'min' | 'max' | 'sum' | 'last';
export type TimeUnit = 'minutes' | 'hours' | 'days' | 'weeks';

export interface DimensionFilter {
  key: string;
  value: string;
}

export interface RankSpec {
  type: RankType;
  value: string;
}

export interface RollupSpec {
  type: RollupType;
  value: number;
  unit: TimeUnit;
}

export interface TimeShiftSpec {
  value: number;
  unit: TimeUnit;
}

export type QueryMode = 'metric' | 'expression' | 'logs';

export type LogFormat = 'logs' | 'table' | 'timeseries';

export type LogOperator =
  | 'is'
  | 'is not'
  | 'is one of'
  | 'is not one of'
  | 'is between'
  | 'is not between'
  | 'exists'
  | 'does not exist';

export interface LogFilter {
  field: string;
  operator: LogOperator;
  values: string[];
}

export interface LogField {
  name: string;
  type: string;
  elasticType: string;
  isAgg: boolean;
}

export interface LogProjectMapping {
  systemField: string;
  portalFields: string[];
}

export interface LogProject {
  id: string;
  name: string;
  projectType?: string;
  extra?: {
    preRenderSystemFields?: string[];
    customMappings?: LogProjectMapping[];
  };
}

export interface VMonitorQuery extends DataQuery {
  queryMode?: QueryMode;
  expression?: string;
  metricName?: string;
  statistic?: StatisticType;
  filters?: DimensionFilter[];
  groupBy?: string[];
  rank?: RankSpec;
  rollup?: RollupSpec;
  rate?: RateType;
  timeShift?: TimeShiftSpec;
  reduce?: ReductionType;
  alias?: string;
  logProjectId?: string;
  logFilters?: LogFilter[];
  logColumns?: string[];
  logGroupBy?: string[];
  logFormat?: LogFormat;
  logLimit?: number;
}

export const DEFAULT_QUERY: Partial<VMonitorQuery> = {
  statistic: 'avg',
  filters: [],
  groupBy: [],
};

export interface Dimension {
  key: string;
  value: string[];
}

export interface InfraResource {
  id: string;
  name: string;
  monitorEnabled: boolean;
}

export const DEFAULT_VMONITOR_URL = 'https://vmonitorapis.vngcloud.vn/vmonitor-api/api';

export const DEFAULT_TOKEN_URL = 'https://iamapis.vngcloud.vn/accounts-api/v2/auth/token';

export interface VMonitorDataSourceOptions extends DataSourceJsonData {
  url?: string;
  clientId?: string;
  tokenUrl?: string;
}

export interface VMonitorSecureJsonData {
  clientSecret?: string;
}
