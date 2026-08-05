import { SelectableValue } from '@grafana/data';
import { LogFormat, LogOperator, RankType, RateType, ReductionType, RollupType, StatisticType, TimeUnit } from './types';

export const STATISTIC_OPTIONS: Array<SelectableValue<StatisticType>> = [
  { label: 'Average', value: 'avg' },
  { label: 'Minimum', value: 'min' },
  { label: 'Maximum', value: 'max' },
  { label: 'Sum', value: 'sum' },
  { label: 'Count', value: 'count' },
  { label: 'Rate (1s)', value: 'rate_1s' },
  { label: 'Rate (1m)', value: 'rate_1m' },
  { label: 'Rate (5m)', value: 'rate_5m' },
];

export const RANK_TYPE_OPTIONS: Array<SelectableValue<RankType>> = [
  { label: 'Top', value: 'topk' },
  { label: 'Bottom', value: 'bottomk' },
];

export const LIMIT_VALUE_OPTIONS: Array<SelectableValue<string>> = ['5', '10', '20', '25', '50'].map((v) => ({
  label: v,
  value: v,
}));

export const ROLLUP_TYPE_OPTIONS: Array<SelectableValue<RollupType>> = [
  { label: 'Average', value: 'avg' },
  { label: 'Minimum', value: 'min' },
  { label: 'Maximum', value: 'max' },
  { label: 'Sum', value: 'sum' },
  { label: 'Count', value: 'count' },
];

export const RATE_OPTIONS: Array<SelectableValue<RateType>> = [
  { label: 'Per second', value: 'per_second' },
  { label: 'Per minute', value: 'per_minute' },
  { label: 'Per hour', value: 'per_hour' },
];

export const REDUCTION_OPTIONS: Array<SelectableValue<ReductionType>> = [
  { label: 'Last', value: 'last' },
  { label: 'Average', value: 'avg' },
  { label: 'Minimum', value: 'min' },
  { label: 'Maximum', value: 'max' },
  { label: 'Sum', value: 'sum' },
];

export const TIME_UNIT_OPTIONS: Array<SelectableValue<TimeUnit>> = [
  { label: 'minutes', value: 'minutes' },
  { label: 'hours', value: 'hours' },
  { label: 'days', value: 'days' },
  { label: 'weeks', value: 'weeks' },
];

export const LOG_FORMAT_OPTIONS: Array<SelectableValue<LogFormat>> = [
  { label: 'Logs', value: 'logs', description: 'Log volume histogram, log lines and expandable detail' },
  { label: 'Table', value: 'table', description: 'One row per log, one column per selected field' },
  { label: 'Time series', value: 'timeseries', description: 'Only the count of logs over time' },
];

export const LOG_OPERATOR_OPTIONS: Array<SelectableValue<LogOperator>> = [
  { label: 'is', value: 'is' },
  { label: 'is not', value: 'is not' },
  { label: 'is one of', value: 'is one of' },
  { label: 'is not one of', value: 'is not one of' },
  { label: 'is between', value: 'is between' },
  { label: 'is not between', value: 'is not between' },
  { label: 'exists', value: 'exists' },
  { label: 'does not exist', value: 'does not exist' },
];

export const LOG_NUMERIC_ELASTIC_TYPES = new Set<string>([
  'long',
  'integer',
  'short',
  'byte',
  'double',
  'float',
  'half_float',
  'scaled_float',
  'date',
]);

export const LOG_MULTI_VALUE_OPERATORS = new Set<LogOperator>(['is one of', 'is not one of']);
export const LOG_RANGE_OPERATORS = new Set<LogOperator>(['is between', 'is not between']);
export const LOG_NO_VALUE_OPERATORS = new Set<LogOperator>(['exists', 'does not exist']);
export const LOG_SUGGESTABLE_OPERATORS = new Set<LogOperator>(['is', 'is not', 'is one of', 'is not one of']);

export const LOG_LIMIT_OPTIONS: Array<SelectableValue<number>> = [50, 100, 200, 500].map((v) => ({
  label: String(v),
  value: v,
}));

export type FunctionId = 'rank' | 'rollup' | 'rate' | 'timeShift' | 'reduce';

export interface FunctionMeta {
  id: FunctionId;
  title: string;
  description: string;
}

export const FUNCTIONS: FunctionMeta[] = [
  { id: 'rollup', title: 'Rollup', description: 'Roll up metrics using an aggregator over a period of time.' },
  { id: 'rate', title: 'Rate', description: 'Graph the rate at which the metric changes.' },
  { id: 'timeShift', title: 'Timeshift', description: 'Compare a metric across different time periods.' },
  {
    id: 'rank',
    title: 'Rank',
    description: 'Keep only the top or bottom N series at each point in time. Series will have gaps.',
  },
  { id: 'reduce', title: 'Reduce', description: 'Collapse each series to a single value (for stat / top-list panels).' },
];
