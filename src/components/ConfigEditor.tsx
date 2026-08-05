import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { VMonitorDataSourceOptions, VMonitorSecureJsonData, DEFAULT_VMONITOR_URL, DEFAULT_TOKEN_URL } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<VMonitorDataSourceOptions, VMonitorSecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onJsonChange = (field: keyof VMonitorDataSourceOptions) => (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: { ...jsonData, [field]: event.target.value },
    });
  };

  const onClientSecretChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: { ...options.secureJsonData, clientSecret: event.target.value },
    });
  };

  const onResetClientSecret = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: { ...options.secureJsonFields, clientSecret: false },
      secureJsonData: { ...options.secureJsonData, clientSecret: '' },
    });
  };

  return (
    <>
      <InlineField
        label="API URL"
        labelWidth={20}
        interactive
        tooltip={
          'vMonitor API base URL. Endpoints such as /v1/statistics are appended to this, and the log search API is derived from the same host.'
        }
      >
        <Input
          id="config-editor-url"
          onChange={onJsonChange('url')}
          value={jsonData.url ?? ''}
          placeholder={DEFAULT_VMONITOR_URL}
          width={60}
        />
      </InlineField>
      <InlineField
        label="Client ID"
        labelWidth={20}
        interactive
        tooltip={'IAM access key id. Combined with the Client Secret it is exchanged for a Bearer token.'}
      >
        <Input
          id="config-editor-client-id"
          onChange={onJsonChange('clientId')}
          value={jsonData.clientId ?? ''}
          placeholder="IAM access key Client ID"
          width={60}
        />
      </InlineField>
      <InlineField
        label="Client Secret"
        labelWidth={20}
        interactive
        tooltip={'IAM access key secret. Stored encrypted (backend only) and exchanged for a Bearer token.'}
      >
        <SecretInput
          required
          id="config-editor-client-secret"
          isConfigured={secureJsonFields?.clientSecret}
          value={secureJsonData?.clientSecret ?? ''}
          placeholder="IAM access key Client Secret"
          width={60}
          onReset={onResetClientSecret}
          onChange={onClientSecretChange}
        />
      </InlineField>
      <InlineField
        label="Token URL"
        labelWidth={20}
        interactive
        tooltip={'IAM token-exchange endpoint. Leave blank to use the default.'}
      >
        <Input
          id="config-editor-token-url"
          onChange={onJsonChange('tokenUrl')}
          value={jsonData.tokenUrl ?? ''}
          placeholder={DEFAULT_TOKEN_URL}
          width={60}
        />
      </InlineField>
    </>
  );
}
