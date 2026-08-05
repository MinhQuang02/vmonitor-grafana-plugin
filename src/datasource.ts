import {
  DataSourceInstanceSettings,
  CoreApp,
  ScopedVars,
  MetricFindValue,
  DataQueryRequest,
  DataQueryResponse,
  DataFrame,
  SupplementaryQueryOptions,
  SupplementaryQueryType,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { TableCellDisplayMode } from '@grafana/ui';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import {
  VMonitorQuery,
  VMonitorDataSourceOptions,
  DEFAULT_QUERY,
  Dimension,
  InfraResource,
  LogField,
  LogProject,
} from './types';
import { InspectValueCell } from './components/InspectValueCell';

export class DataSource extends DataSourceWithBackend<VMonitorQuery, VMonitorDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<VMonitorDataSourceOptions>) {
    super(instanceSettings);
  }

  query(request: DataQueryRequest<VMonitorQuery>): Observable<DataQueryResponse> {
    return super.query(request).pipe(
      map((response) => ({
        ...response,
        data: response.data?.map(attachInspectValueCellRenderer) ?? [],
      }))
    );
  }

  getDefaultQuery(_: CoreApp): Partial<VMonitorQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: VMonitorQuery, scopedVars: ScopedVars): VMonitorQuery {
    const srv = getTemplateSrv();
    const replace = (v?: string) => (v ? srv.replace(v, scopedVars) : v);
    return {
      ...query,
      metricName: replace(query.metricName),
      expression: replace(query.expression),
      alias: replace(query.alias),
      filters: query.filters?.map((f) => ({ key: f.key, value: srv.replace(f.value, scopedVars) })),
      groupBy: query.groupBy?.map((g) => srv.replace(g, scopedVars)),
      logProjectId: replace(query.logProjectId),
      logGroupBy: query.logGroupBy?.map((g) => srv.replace(g, scopedVars)),
      logFilters: query.logFilters?.map((f) => ({
        ...f,
        values: f.values?.map((v) => srv.replace(v, scopedVars)) ?? [],
      })),
    };
  }

  filterQuery(query: VMonitorQuery): boolean {
    if (query.queryMode === 'logs') {
      return !!query.logProjectId;
    }
    if (query.queryMode === 'expression') {
      return !!query.expression;
    }
    return !!query.metricName;
  }

  getSupportedSupplementaryQueryTypes(): SupplementaryQueryType[] {
    return [SupplementaryQueryType.LogsVolume];
  }

  getSupplementaryQuery(options: SupplementaryQueryOptions, query: VMonitorQuery): VMonitorQuery | undefined {
    if (options.type !== SupplementaryQueryType.LogsVolume || query.queryMode !== 'logs' || !query.logProjectId) {
      return undefined;
    }
    return { ...query, logFormat: 'timeseries', logGroupBy: [], refId: `${query.refId}-logs-volume` };
  }

  getSupplementaryRequest(
    type: SupplementaryQueryType,
    request: DataQueryRequest<VMonitorQuery>,
    options?: SupplementaryQueryOptions
  ): DataQueryRequest<VMonitorQuery> | undefined {
    if (type !== SupplementaryQueryType.LogsVolume) {
      return undefined;
    }
    const targets = request.targets
      .map((query) => this.getSupplementaryQuery({ type, ...options } as SupplementaryQueryOptions, query))
      .filter((query): query is VMonitorQuery => !!query);
    if (targets.length === 0) {
      return undefined;
    }
    return { ...request, targets };
  }

  async getLogProjects(): Promise<LogProject[]> {
    return this.getResource('log-projects');
  }

  async getLogMappings(project: string): Promise<LogField[]> {
    if (!project) {
      return [];
    }
    return this.getResource('log-mappings', { project });
  }

  async getLogValues(project: string, field: string, query = ''): Promise<string[]> {
    if (!project || !field) {
      return [];
    }
    return this.getResource('log-values', { project, field, query });
  }

  async getMetricNames(): Promise<string[]> {
    return this.getResource('metric-names');
  }

  async getDimensions(metric: string): Promise<Dimension[]> {
    if (!metric) {
      return [];
    }
    return this.getResource('dimensions', { metric });
  }

  async getDimensionNames(): Promise<string[]> {
    return this.getResource('dimension-names');
  }

  async getInfrastructure(product: string, search = '', region = ''): Promise<InfraResource[]> {
    if (!product) {
      return [];
    }
    return this.getResource('infrastructure', { product, search, region });
  }

  async getInfraRegions(product: string): Promise<string[]> {
    if (!product) {
      return [];
    }
    return this.getResource('infrastructure-regions', { product });
  }

  async metricFindQuery(query: string): Promise<MetricFindValue[]> {
    const raw = getTemplateSrv().replace(query ?? '').trim();

    const metricsMatch = raw.match(/^metrics\(\s*\)$/i);
    if (metricsMatch) {
      return toValues(await this.getMetricNames());
    }

    const dimsMatch = raw.match(/^dimensions\(\s*([^)]+?)\s*\)$/i);
    if (dimsMatch) {
      const dims = await this.getDimensions(dimsMatch[1]);
      return toValues(dims.map((d) => d.key));
    }

    const valuesMatch = raw.match(/^dimensionValues\(\s*([^,]+?)\s*,\s*([^)]+?)\s*\)$/i);
    if (valuesMatch) {
      const [, metric, key] = valuesMatch;
      const dims = await this.getDimensions(metric);
      const found = dims.find((d) => d.key === key.trim());
      return toValues(found?.value ?? []);
    }

    const infraMatch = raw.match(/^infrastructure\(\s*([^,)]+?)\s*(?:,\s*([^)]*?)\s*)?\)$/i);
    if (infraMatch) {
      const [, product, region] = infraMatch;
      const resources = await this.getInfrastructure(product, '', region ?? '');
      return resources.map((r) => ({ text: r.name, value: r.id }));
    }

    const regionsMatch = raw.match(/^regions\(\s*([^)]+?)\s*\)$/i);
    if (regionsMatch) {
      return toValues(await this.getInfraRegions(regionsMatch[1]));
    }

    return [];
  }
}

function toValues(items: string[]): MetricFindValue[] {
  return items.map((text) => ({ text }));
}

function attachInspectValueCellRenderer(frame: DataFrame): DataFrame {
  const inspectField = frame.fields.find((field) => field.name === 'inspect');
  if (!inspectField) {
    return frame;
  }

  inspectField.config = {
    ...inspectField.config,
    custom: {
      ...inspectField.config?.custom,
      align: 'center',
      cellOptions: {
        type: TableCellDisplayMode.Custom,
        cellComponent: InspectValueCell,
      },
      hideHeader: true,
      width: 52,
      minWidth: 52,
    },
  };

  return frame;
}
