import React, { useMemo, useState } from 'react';
import { CustomCellRendererProps, IconButton, TableCellInspector, TableCellInspectorMode } from '@grafana/ui';

export function InspectValueCell({ value }: CustomCellRendererProps) {
  const [isOpen, setIsOpen] = useState(false);
  const inspectValue = useMemo(() => toInspectText(value), [value]);

  return (
    <div style={{ display: 'flex', justifyContent: 'center', width: '100%' }}>
      <IconButton name="eye" size="sm" tooltip="Inspect value" onClick={() => setIsOpen(true)} aria-label="Inspect value" />
      {isOpen && (
        <TableCellInspector
          value={inspectValue}
          mode={TableCellInspectorMode.code}
          onDismiss={() => setIsOpen(false)}
        />
      )}
    </div>
  );
}

function toInspectText(value: unknown): string {
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return '';
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return value;
    }
  }
  if (value === null || value === undefined) {
    return '';
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
