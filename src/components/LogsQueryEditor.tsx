import React, { useEffect, useMemo, useState } from 'react';
import { CoreApp, QueryEditorProps, SelectableValue } from '@grafana/data';
import { Button, IconButton, Input, MultiSelect, RadioButtonGroup, Select, Stack } from '@grafana/ui';
import { DataSource } from '../datasource';
import { LogField, LogFilter, LogFormat, LogOperator, LogProject, VMonitorDataSourceOptions, VMonitorQuery } from '../types';
import {
  LOG_FORMAT_OPTIONS,
  LOG_LIMIT_OPTIONS,
  LOG_MULTI_VALUE_OPERATORS,
  LOG_NO_VALUE_OPERATORS,
  LOG_OPERATOR_OPTIONS,
  LOG_RANGE_OPERATORS,
  LOG_SUGGESTABLE_OPERATORS,
} from '../constants';
import { EditorField, EditorFieldGroup, EditorRow, EditorRows, Hint } from './editor/Editor';

type Props = QueryEditorProps<DataSource, VMonitorQuery, VMonitorDataSourceOptions>;

function isAlertingApp(app?: CoreApp): boolean {
  return app === CoreApp.UnifiedAlerting || app === CoreApp.CloudAlerting;
}

function toOption(value: string): SelectableValue<string> {
  return { label: value, value };
}

function mergeOptions(options: Array<SelectableValue<string>>, ...selected: string[]): Array<SelectableValue<string>> {
  const merged = [...options];
  for (const value of selected) {
    if (value && !merged.some((o) => o.value === value)) {
      merged.unshift(toOption(value));
    }
  }
  return merged;
}

function resolveDefaultColumns(project: LogProject | undefined, mappings: LogField[]): string[] {
  const present = new Set(mappings.filter((f) => f.elasticType !== 'keyword').map((f) => f.name));
  const preRender = project?.extra?.preRenderSystemFields ?? [];
  const systemToPortal = new Map<string, string>();
  const portalToSystem = new Map<string, string[]>();
  for (const mapping of project?.extra?.customMappings ?? []) {
    const portal = mapping.portalFields?.[0];
    if (!portal) {
      continue;
    }
    systemToPortal.set(mapping.systemField, portal);
    portalToSystem.set(portal, [...(portalToSystem.get(portal) ?? []), mapping.systemField]);
  }

  const columns: string[] = [];
  for (const field of preRender) {
    if (field.toLowerCase() === 'severity') {
      continue;
    }
    if (present.has(field)) {
      columns.push(field);
      continue;
    }
    const portal = systemToPortal.get(field);
    const alternative = portal ? (portalToSystem.get(portal) ?? []).find((s) => present.has(s)) : undefined;
    if (alternative && alternative.toLowerCase() !== 'severity' && !columns.includes(alternative)) {
      columns.push(alternative);
    }
  }
  return columns.length > 0 ? columns : ['@timestamp'];
}

export function LogsQueryEditor({ datasource, query, onChange, onRunQuery, app }: Props) {
  const [projects, setProjects] = useState<LogProject[]>([]);
  const [fields, setFields] = useState<LogField[]>([]);

  const alerting = isAlertingApp(app);

  useEffect(() => {
    if (alerting && query.logFormat !== 'timeseries') {
      onChange({ ...query, logFormat: 'timeseries' });
    }
  }, [alerting, query, onChange]);

  useEffect(() => {
    let active = true;
    datasource
      .getLogProjects()
      .then((list) => active && setProjects(list))
      .catch(() => active && setProjects([]));
    return () => {
      active = false;
    };
  }, [datasource]);

  useEffect(() => {
    let active = true;
    datasource
      .getLogMappings(query.logProjectId ?? '')
      .then((list) => active && setFields(list))
      .catch(() => active && setFields([]));
    return () => {
      active = false;
    };
  }, [datasource, query.logProjectId]);

  const patch = (partial: Partial<VMonitorQuery>, run = true) => {
    onChange({ ...query, ...partial });
    if (run) {
      onRunQuery();
    }
  };

  const projectOptions = useMemo<Array<SelectableValue<string>>>(
    () => projects.map((p) => ({ label: p.name, value: p.id, description: p.projectType })),
    [projects]
  );

  const fieldByName = useMemo(() => new Map(fields.map((f) => [f.name, f])), [fields]);

  const filterFieldOptions = useMemo<Array<SelectableValue<string>>>(
    () => fields.map((f) => ({ label: f.name, value: f.name, description: f.elasticType })),
    [fields]
  );

  const columnFieldOptions = useMemo<Array<SelectableValue<string>>>(
    () =>
      fields
        .filter((f) => f.elasticType !== 'keyword')
        .map((f) => ({ label: f.name, value: f.name })),
    [fields]
  );

  const groupByFieldOptions = useMemo<Array<SelectableValue<string>>>(
    () => fields.filter((f) => f.isAgg).map((f) => ({ label: f.name, value: f.name, description: f.elasticType })),
    [fields]
  );

  const onProjectChange = async (sel: SelectableValue<string>) => {
    const projectId = sel.value;
    const base: VMonitorQuery = { ...query, logProjectId: projectId, logFilters: [], logGroupBy: [] };
    onChange({ ...base, logColumns: [] });
    if (!projectId) {
      onRunQuery();
      return;
    }
    let columns: string[] = [];
    try {
      const mappings = await datasource.getLogMappings(projectId);
      columns = resolveDefaultColumns(projects.find((p) => p.id === projectId), mappings);
    } catch {
      columns = ['@timestamp'];
    }
    onChange({ ...base, logColumns: columns });
    onRunQuery();
  };

  const filters = query.logFilters ?? [];
  const setFilters = (next: LogFilter[]) => patch({ logFilters: next });
  const addFilter = () => setFilters([...filters, { field: '', operator: 'is', values: [] }]);
  const removeFilter = (i: number) => setFilters(filters.filter((_, idx) => idx !== i));
  const updateFilter = (i: number, partial: Partial<LogFilter>) =>
    setFilters(filters.map((f, idx) => (idx === i ? { ...f, ...partial } : f)));

  const onOperatorChange = (i: number, operator: LogOperator) => {
    const keepValues = LOG_RANGE_OPERATORS.has(operator) || LOG_MULTI_VALUE_OPERATORS.has(operator);
    updateFilter(i, { operator, values: keepValues ? filters[i].values ?? [] : [] });
  };

  const columns = query.logColumns ?? [];
  const setColumns = (next: string[]) => patch({ logColumns: next });
  const addColumn = () => setColumns([...columns, '']);
  const updateColumn = (i: number, value: string) => setColumns(columns.map((c, idx) => (idx === i ? value : c)));
  const removeColumn = (i: number) => setColumns(columns.filter((_, idx) => idx !== i));
  const moveColumn = (i: number, direction: number) => {
    const target = i + direction;
    if (target < 0 || target >= columns.length) {
      return;
    }
    const next = [...columns];
    [next[i], next[target]] = [next[target], next[i]];
    setColumns(next);
  };

  const format: LogFormat = alerting ? 'timeseries' : query.logFormat ?? 'logs';

  const groupBy = query.logGroupBy ?? [];

  return (
    <EditorRows>
      <EditorRow>
        <EditorFieldGroup>
          <EditorField label="Log project" grow>
            <Select
              inputId="vm-log-project"
              options={projectOptions}
              value={query.logProjectId ?? null}
              onChange={onProjectChange}
              placeholder="Select log project"
              isClearable
              width={40}
            />
          </EditorField>
          {!alerting && (
            <EditorField label="Format">
              <RadioButtonGroup options={LOG_FORMAT_OPTIONS} value={format} onChange={(v) => patch({ logFormat: v })} />
            </EditorField>
          )}
          {!alerting && format !== 'timeseries' && (
            <EditorField label="Limit">
              <Select
                inputId="vm-log-limit"
                options={LOG_LIMIT_OPTIONS}
                value={query.logLimit ?? 100}
                onChange={(sel) => patch({ logLimit: sel.value ?? 100 })}
                width={12}
              />
            </EditorField>
          )}
        </EditorFieldGroup>
        {alerting && (
          <Hint icon="info-circle">
            An alert rule evaluates the number of matching logs over time. Add a Reduce expression (use Sum to count
            every log in the range, or Last for the most recent bucket) and then a Threshold expression.
          </Hint>
        )}
      </EditorRow>

      <EditorRow label="Filters">
        {filters.map((f, i) => (
          <Stack key={i} direction="row" gap={0.5} alignItems="center">
            <Select
              inputId={`vm-log-filter-field-${i}`}
              options={
                f.field ? [toOption(f.field), ...filterFieldOptions.filter((o) => o.value !== f.field)] : filterFieldOptions
              }
              value={f.field || null}
              onChange={(sel) => updateFilter(i, { field: sel.value ?? '', values: [] })}
              placeholder="field"
              allowCustomValue
              width={32}
            />
            <Select
              inputId={`vm-log-filter-op-${i}`}
              options={LOG_OPERATOR_OPTIONS}
              value={f.operator}
              onChange={(sel) => onOperatorChange(i, (sel.value as LogOperator) ?? 'is')}
              width={18}
            />
            <LogFilterValue
              index={i}
              filter={f}
              datasource={datasource}
              projectId={query.logProjectId ?? ''}
              isAgg={!!fieldByName.get(f.field)?.isAgg}
              onChange={(values) => updateFilter(i, { values })}
              onRunQuery={onRunQuery}
            />
            <IconButton name="trash-alt" tooltip="Remove filter" onClick={() => removeFilter(i)} />
          </Stack>
        ))}
        <div>
          <Button variant="secondary" size="sm" icon="plus" onClick={addFilter} disabled={!query.logProjectId}>
            Add filter
          </Button>
        </div>
      </EditorRow>

      {format === 'timeseries' && (
        <EditorRow>
          <EditorField label="Group by" grow>
            <MultiSelect
              inputId="vm-log-groupby"
              options={mergeOptions(groupByFieldOptions, ...groupBy)}
              value={groupBy}
              onChange={(vals) => patch({ logGroupBy: vals.map((v) => v.value!).filter(Boolean) })}
              placeholder="none — one series for all matching logs"
              allowCustomValue
              isDisabled={!query.logProjectId}
              width={64}
            />
          </EditorField>
          {groupBy.length > 0 && (
            <Hint icon={alerting ? 'exclamation-triangle' : 'info-circle'}>
              {alerting
                ? 'Each group value becomes its own series, and so its own alert instance. A group that has no logs at all is absent from the response and cannot alert — to catch logs stopping entirely, use a separate rule with no group by.'
                : 'Splits the count into one series per value, at most 20 values per field.'}
            </Hint>
          )}
        </EditorRow>
      )}

      {format !== 'timeseries' && (
        <EditorRow label="Columns">
          {columns.map((col, i) => (
            <Stack key={i} direction="row" gap={0.5} alignItems="center">
              <Select
                inputId={`vm-log-column-${i}`}
                options={col ? mergeOptions(columnFieldOptions, col) : columnFieldOptions}
                value={col || null}
                onChange={(sel) => updateColumn(i, sel.value ?? '')}
                placeholder="field"
                allowCustomValue
                width={40}
              />
              <IconButton
                name="arrow-up"
                tooltip="Move up"
                disabled={i === 0}
                onClick={() => moveColumn(i, -1)}
              />
              <IconButton
                name="arrow-down"
                tooltip="Move down"
                disabled={i === columns.length - 1}
                onClick={() => moveColumn(i, 1)}
              />
              <IconButton name="trash-alt" tooltip="Remove column" onClick={() => removeColumn(i)} />
            </Stack>
          ))}
          <div>
            <Button variant="secondary" size="sm" icon="plus" onClick={addColumn} disabled={!query.logProjectId}>
              Add column
            </Button>
          </div>
        </EditorRow>
      )}
    </EditorRows>
  );
}

function LogFilterValue({
  index,
  filter,
  datasource,
  projectId,
  isAgg,
  onChange,
  onRunQuery,
}: {
  index: number;
  filter: LogFilter;
  datasource: DataSource;
  projectId: string;
  isAgg: boolean;
  onChange: (values: string[]) => void;
  onRunQuery: () => void;
}) {
  const values = filter.values ?? [];
  const [suggestions, setSuggestions] = useState<Array<SelectableValue<string>>>([]);
  const suggestable = isAgg && LOG_SUGGESTABLE_OPERATORS.has(filter.operator);

  useEffect(() => {
    if (!suggestable || !projectId || !filter.field) {
      return;
    }
    let active = true;
    datasource
      .getLogValues(projectId, filter.field)
      .then((vals) => active && setSuggestions(vals.map(toOption)))
      .catch(() => active && setSuggestions([]));
    return () => {
      active = false;
    };
  }, [datasource, projectId, filter.field, suggestable]);

  if (LOG_NO_VALUE_OPERATORS.has(filter.operator)) {
    return null;
  }

  if (LOG_RANGE_OPERATORS.has(filter.operator)) {
    return (
      <Stack direction="row" gap={0.5} alignItems="center">
        <Input
          id={`vm-log-filter-from-${index}`}
          value={values[0] ?? ''}
          onChange={(e) => onChange([e.currentTarget.value, values[1] ?? ''])}
          onBlur={onRunQuery}
          placeholder="from"
          width={16}
        />
        <Input
          id={`vm-log-filter-to-${index}`}
          value={values[1] ?? ''}
          onChange={(e) => onChange([values[0] ?? '', e.currentTarget.value])}
          onBlur={onRunQuery}
          placeholder="to"
          width={16}
        />
      </Stack>
    );
  }

  if (LOG_MULTI_VALUE_OPERATORS.has(filter.operator)) {
    return (
      <MultiSelect
        inputId={`vm-log-filter-values-${index}`}
        options={mergeOptions(suggestions, ...values)}
        value={values.map(toOption)}
        onChange={(vals) => onChange(vals.map((v) => v.value!).filter(Boolean))}
        placeholder={isAgg ? 'select or type values' : 'type values'}
        allowCustomValue
        noOptionsMessage="Type a value and press enter"
        width={32}
      />
    );
  }

  if (isAgg) {
    return (
      <Select
        inputId={`vm-log-filter-value-${index}`}
        options={mergeOptions(suggestions, values[0] ?? '')}
        value={values[0] || null}
        onChange={(sel) => onChange(sel.value ? [sel.value] : [])}
        placeholder="select or type value"
        allowCustomValue
        width={32}
      />
    );
  }

  return (
    <Input
      id={`vm-log-filter-value-${index}`}
      value={values[0] ?? ''}
      onChange={(e) => onChange([e.currentTarget.value])}
      onBlur={onRunQuery}
      placeholder="value"
      width={32}
    />
  );
}
