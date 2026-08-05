import React, { useEffect, useMemo, useState } from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { Button, IconButton, Input, MultiSelect, RadioButtonGroup, Select, Stack, Text } from '@grafana/ui';
import { DataSource } from '../datasource';
import { LogsQueryEditor } from './LogsQueryEditor';
import { EditorContainer, EditorField, EditorFieldGroup, EditorHeader, EditorRow, EditorRows, Hint } from './editor/Editor';
import {
  DimensionFilter,
  VMonitorDataSourceOptions,
  VMonitorQuery,
  QueryMode,
  RankType,
  RateType,
  ReductionType,
  RollupType,
  StatisticType,
  TimeUnit,
} from '../types';
import {
  FUNCTIONS,
  FunctionId,
  LIMIT_VALUE_OPTIONS,
  RANK_TYPE_OPTIONS,
  RATE_OPTIONS,
  REDUCTION_OPTIONS,
  ROLLUP_TYPE_OPTIONS,
  STATISTIC_OPTIONS,
  TIME_UNIT_OPTIONS,
} from '../constants';

type Props = QueryEditorProps<DataSource, VMonitorQuery, VMonitorDataSourceOptions>;

type TopMode = 'metric' | 'logs';

const TOP_MODE_OPTIONS: Array<SelectableValue<TopMode>> = [
  { label: 'Metric', value: 'metric' },
  { label: 'Logs', value: 'logs' },
];

const MODE_OPTIONS: Array<SelectableValue<QueryMode>> = [
  { label: 'Metric', value: 'metric' },
  { label: 'Expression', value: 'expression' },
];

const EXPRESSION_FUNCTION_IDS = new Set(['rank', 'timeShift', 'reduce']);

const RATE_STATISTICS = new Set<string>(['rate_1s', 'rate_1m', 'rate_5m']);

function withCurrentValue(options: Array<SelectableValue<string>>, value?: string): Array<SelectableValue<string>> {
  if (!value || options.some((o) => o.value === value)) {
    return options;
  }
  return [{ label: value, value }, ...options];
}

function isUnlisted(options: Array<SelectableValue<string>>, value?: string): boolean {
  return !!value && !value.startsWith('$') && options.length > 0 && !options.some((o) => o.value === value);
}

export function QueryEditor({ datasource, query, onChange, onRunQuery, app }: Props) {
  const [metricOptions, setMetricOptions] = useState<Array<SelectableValue<string>>>([]);
  const [dimensions, setDimensions] = useState<Array<{ key: string; value: string[] }>>([]);

  useEffect(() => {
    let active = true;
    datasource
      .getMetricNames()
      .then((names) => active && setMetricOptions(names.map((n) => ({ label: n, value: n }))))
      .catch(() => active && setMetricOptions([]));
    return () => {
      active = false;
    };
  }, [datasource]);

  useEffect(() => {
    let active = true;
    datasource
      .getDimensions(query.metricName ?? '')
      .then((dims) => active && setDimensions(dims.map((d) => ({ key: d.key, value: d.value ?? [] }))))
      .catch(() => active && setDimensions([]));
    return () => {
      active = false;
    };
  }, [datasource, query.metricName]);

  const dimensionKeyOptions = useMemo<Array<SelectableValue<string>>>(
    () => dimensions.map((d) => ({ label: d.key, value: d.key })),
    [dimensions]
  );

  const valueOptionsFor = (key: string): Array<SelectableValue<string>> => {
    const found = dimensions.find((d) => d.key === key);
    return (found?.value ?? []).map((v) => ({ label: v, value: v }));
  };

  const patch = (partial: Partial<VMonitorQuery>, run = true) => {
    onChange({ ...query, ...partial });
    if (run) {
      onRunQuery();
    }
  };

  const mode: QueryMode = query.queryMode ?? 'metric';
  const topMode: TopMode = mode === 'logs' ? 'logs' : 'metric';

  const onTopModeChange = (next: TopMode) => {
    if (next === topMode) {
      return;
    }
    patch({ queryMode: next === 'logs' ? 'logs' : 'metric' });
  };

  const onMetricChange = (sel: SelectableValue<string>) => {
    patch({ metricName: sel.value, filters: [], groupBy: [] });
  };

  const filters = query.filters ?? [];
  const setFilters = (next: DimensionFilter[]) => patch({ filters: next });
  const addFilter = () => setFilters([...filters, { key: '', value: '' }]);
  const updateFilter = (i: number, partial: Partial<DimensionFilter>) =>
    setFilters(filters.map((f, idx) => (idx === i ? { ...f, ...partial } : f)));
  const removeFilter = (i: number) => setFilters(filters.filter((_, idx) => idx !== i));

  const enabled = new Set<FunctionId>();
  if (query.rank) {
    enabled.add('rank');
  }
  if (query.rollup) {
    enabled.add('rollup');
  }
  if (query.rate) {
    enabled.add('rate');
  }
  if (query.timeShift) {
    enabled.add('timeShift');
  }
  if (query.reduce) {
    enabled.add('reduce');
  }

  const addFunction = (id: FunctionId) => {
    switch (id) {
      case 'rank':
        return patch({ rank: { type: 'topk', value: '5' } });
      case 'rollup':
        return patch({ rollup: { type: 'avg', value: 1, unit: 'minutes' } });
      case 'rate':
        return patch({ rate: 'per_second' });
      case 'timeShift':
        return patch({ timeShift: { value: 5, unit: 'minutes' } });
      case 'reduce':
        return patch({ reduce: 'last' });
    }
  };

  const removeFunction = (id: FunctionId) => {
    const clear: Record<FunctionId, Partial<VMonitorQuery>> = {
      rank: { rank: undefined },
      rollup: { rollup: undefined },
      rate: { rate: undefined },
      timeShift: { timeShift: undefined },
      reduce: { reduce: undefined },
    };
    patch(clear[id]);
  };

  const availableFunctions = FUNCTIONS.filter(
    (f) => !enabled.has(f.id) && (mode === 'metric' || EXPRESSION_FUNCTION_IDS.has(f.id))
  ).map((f) => ({
    label: f.title,
    value: f.id,
    description: f.description,
  }));

  return (
    <EditorContainer>
      <EditorHeader
        right={<RadioButtonGroup size="sm" options={TOP_MODE_OPTIONS} value={topMode} onChange={onTopModeChange} />}
      />

      {topMode === 'logs' && (
        <LogsQueryEditor datasource={datasource} query={query} onChange={onChange} onRunQuery={onRunQuery} app={app} />
      )}

      {topMode === 'metric' && (
        <EditorRows>
          <EditorRow>
            <EditorFieldGroup>
              <EditorField label="Query type">
                <RadioButtonGroup options={MODE_OPTIONS} value={mode} onChange={(v) => patch({ queryMode: v })} />
              </EditorField>
              {mode === 'metric' && (
                <EditorField label="Metric" grow>
                  <Select
                    inputId="vm-metric"
                    options={withCurrentValue(metricOptions, query.metricName)}
                    value={query.metricName ?? null}
                    onChange={onMetricChange}
                    placeholder="Select metric"
                    isClearable
                    allowCustomValue
                    width={40}
                  />
                </EditorField>
              )}
              {mode === 'metric' && (
                <EditorField label="Statistic">
                  <Select
                    inputId="vm-statistic"
                    options={STATISTIC_OPTIONS}
                    value={query.statistic ?? 'avg'}
                    onChange={(s) => patch({ statistic: s.value as StatisticType })}
                    width={20}
                  />
                </EditorField>
              )}
              {mode === 'expression' && (
                <EditorField label="Expression" grow>
                  <Input
                    id="vm-expression"
                    value={query.expression ?? ''}
                    onChange={(e) => patch({ expression: e.currentTarget.value }, false)}
                    onBlur={onRunQuery}
                    placeholder="A / B * 100"
                    width={50}
                  />
                </EditorField>
              )}
            </EditorFieldGroup>
            {mode === 'metric' && RATE_STATISTICS.has(query.statistic ?? '') && (
              <Hint icon="info-circle">
                Rate statistics are served by a different vMonitor endpoint that does not support Rollup, Rate or
                Timeshift, and cannot be used inside an Expression.
              </Hint>
            )}
            {mode === 'metric' && isUnlisted(metricOptions, query.metricName) && (
              <Hint icon="info-circle">
                This metric is not offered by vMonitor for the current selection — usually because detailed monitoring is
                off. The query still runs.
              </Hint>
            )}
          </EditorRow>

          {mode === 'metric' && (
            <EditorRow label="Filter by">
              {filters.map((f, i) => (
                <Stack key={i} direction="row" gap={0.5} alignItems="center">
                  <Select
                    inputId={`vm-filter-key-${i}`}
                    options={withCurrentValue(dimensionKeyOptions, f.key)}
                    value={f.key || null}
                    onChange={(sel) => updateFilter(i, { key: sel.value ?? '', value: '' })}
                    placeholder="dimension"
                    allowCustomValue
                    width={28}
                  />
                  <Text variant="bodySmall" color="secondary">
                    =
                  </Text>
                  <Select
                    inputId={`vm-filter-val-${i}`}
                    options={withCurrentValue(valueOptionsFor(f.key), f.value)}
                    value={f.value || null}
                    onChange={(sel) => updateFilter(i, { value: sel.value ?? '' })}
                    placeholder="value or $variable"
                    allowCustomValue
                    width={28}
                  />
                  <IconButton name="trash-alt" tooltip="Remove filter" onClick={() => removeFilter(i)} />
                </Stack>
              ))}
              {filters.some((f) => isUnlisted(dimensionKeyOptions, f.key)) && (
                <Hint icon="info-circle">
                  This dimension is not offered by vMonitor for the current selection — usually because detailed
                  monitoring is off. The query still runs.
                </Hint>
              )}
              <div>
                <Button variant="secondary" size="sm" icon="plus" onClick={addFilter}>
                  Add filter
                </Button>
              </div>
            </EditorRow>
          )}

          {mode === 'metric' && (
            <EditorRow>
              <EditorField label="Group by" grow>
                <MultiSelect
                  inputId="vm-groupby"
                  options={dimensionKeyOptions}
                  value={query.groupBy ?? []}
                  onChange={(vals) => patch({ groupBy: vals.map((v) => v.value!).filter(Boolean) })}
                  placeholder="none"
                  allowCustomValue
                  width={64}
                />
              </EditorField>
            </EditorRow>
          )}

          {enabled.size > 0 && (
            <EditorRow label="Functions">
              {mode === 'metric' && enabled.has('rollup') && query.rollup && (
                <FunctionRow title="Rollup" onRemove={() => removeFunction('rollup')}>
                  <Select
                    width={16}
                    options={ROLLUP_TYPE_OPTIONS}
                    value={query.rollup.type}
                    onChange={(s) => patch({ rollup: { ...query.rollup!, type: s.value as RollupType } })}
                  />
                  <Input
                    width={10}
                    type="number"
                    value={query.rollup.value}
                    onChange={(e) =>
                      patch({ rollup: { ...query.rollup!, value: e.currentTarget.valueAsNumber || 0 } }, false)
                    }
                    onBlur={onRunQuery}
                  />
                  <Select
                    width={14}
                    options={TIME_UNIT_OPTIONS}
                    value={query.rollup.unit}
                    onChange={(s) => patch({ rollup: { ...query.rollup!, unit: s.value as TimeUnit } })}
                  />
                </FunctionRow>
              )}

              {mode === 'metric' && enabled.has('rate') && query.rate && (
                <FunctionRow title="Rate" onRemove={() => removeFunction('rate')}>
                  <Select
                    width={18}
                    options={RATE_OPTIONS}
                    value={query.rate}
                    onChange={(s) => patch({ rate: s.value as RateType })}
                  />
                </FunctionRow>
              )}

              {enabled.has('timeShift') && query.timeShift && (
                <FunctionRow title="Timeshift" onRemove={() => removeFunction('timeShift')}>
                  <Input
                    width={10}
                    type="number"
                    value={query.timeShift.value}
                    onChange={(e) =>
                      patch({ timeShift: { ...query.timeShift!, value: e.currentTarget.valueAsNumber || 0 } }, false)
                    }
                    onBlur={onRunQuery}
                  />
                  <Select
                    width={14}
                    options={TIME_UNIT_OPTIONS}
                    value={query.timeShift.unit}
                    onChange={(s) => patch({ timeShift: { ...query.timeShift!, unit: s.value as TimeUnit } })}
                  />
                </FunctionRow>
              )}

              {enabled.has('rank') && query.rank && (
                <>
                  <FunctionRow title="Rank" onRemove={() => removeFunction('rank')}>
                    <Select
                      width={14}
                      options={RANK_TYPE_OPTIONS}
                      value={query.rank.type}
                      onChange={(s) => patch({ rank: { ...query.rank!, type: s.value as RankType } })}
                    />
                    <Select
                      width={12}
                      options={LIMIT_VALUE_OPTIONS}
                      value={query.rank.value}
                      onChange={(s) => patch({ rank: { ...query.rank!, value: s.value ?? '5' } })}
                      allowCustomValue
                    />
                  </FunctionRow>
                  <Hint icon="exclamation-triangle">
                    Rank picks the top/bottom N series at every data point, so each series ends up with gaps. In an alert
                    rule, put a Resample expression before Reduce, otherwise Reduce compares values from different points
                    in time.
                  </Hint>
                </>
              )}

              {enabled.has('reduce') && query.reduce && (
                <FunctionRow title="Reduce" onRemove={() => removeFunction('reduce')}>
                  <Select
                    width={16}
                    options={REDUCTION_OPTIONS}
                    value={query.reduce}
                    onChange={(s) => patch({ reduce: s.value as ReductionType })}
                  />
                </FunctionRow>
              )}
            </EditorRow>
          )}

          <EditorRow>
            <EditorFieldGroup>
              <EditorField label="Add function">
                <Select
                  inputId="vm-add-function"
                  options={availableFunctions}
                  value={null}
                  onChange={(s) => s.value && addFunction(s.value)}
                  placeholder="Add function"
                  isDisabled={availableFunctions.length === 0}
                  width={24}
                />
              </EditorField>
              <EditorField label="Alias" grow>
                <Input
                  value={query.alias ?? ''}
                  onChange={(e) => patch({ alias: e.currentTarget.value }, false)}
                  onBlur={onRunQuery}
                  placeholder="{{host}}"
                  width={30}
                />
              </EditorField>
            </EditorFieldGroup>
          </EditorRow>
        </EditorRows>
      )}
    </EditorContainer>
  );
}

function FunctionRow({ title, onRemove, children }: { title: string; onRemove: () => void; children: React.ReactNode }) {
  return (
    <Stack direction="row" gap={0.5} alignItems="center">
      <div style={{ minWidth: 84 }}>
        <Text variant="bodySmall" color="secondary">
          {title}
        </Text>
      </div>
      {children}
      <IconButton name="trash-alt" tooltip={`Remove ${title}`} onClick={onRemove} />
    </Stack>
  );
}
