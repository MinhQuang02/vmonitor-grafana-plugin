import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Field, Icon, Stack, Text, useStyles2 } from '@grafana/ui';

const getStyles = (theme: GrafanaTheme2) => ({
  container: css({
    paddingTop: theme.spacing(1),
    marginLeft: theme.spacing(-3),
  }),
  header: css({
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(1),
  }),
  headerLeft: css({
    flex: 1,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  }),
  rows: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
  }),
  row: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
    background: theme.colors.background.secondary,
    borderRadius: theme.shape.radius.default,
    padding: theme.spacing(1),
  }),
  rowLabel: css({
    fontSize: theme.typography.bodySmall.fontSize,
    fontWeight: theme.typography.fontWeightMedium,
    color: theme.colors.text.secondary,
  }),
});

export function Hint({
  icon,
  children,
}: {
  icon: 'info-circle' | 'exclamation-triangle';
  children: NonNullable<React.ReactNode>;
}) {
  return (
    <Stack direction="row" gap={0.5} alignItems="center">
      <Icon name={icon} />
      <Text variant="bodySmall" color="secondary">
        {children}
      </Text>
    </Stack>
  );
}

export function EditorContainer({ children }: { children: React.ReactNode }) {
  const styles = useStyles2(getStyles);
  return <div className={styles.container}>{children}</div>;
}

export function EditorHeader({ left, right }: { left?: React.ReactNode; right?: React.ReactNode }) {
  const styles = useStyles2(getStyles);
  return (
    <div className={styles.header}>
      <div className={styles.headerLeft}>{left}</div>
      {right}
    </div>
  );
}

export function EditorRows({ children }: { children: React.ReactNode }) {
  const styles = useStyles2(getStyles);
  return <div className={styles.rows}>{children}</div>;
}

export function EditorRow({ label, children }: { label?: string; children: React.ReactNode }) {
  const styles = useStyles2(getStyles);
  return (
    <div className={styles.row}>
      {label && <div className={styles.rowLabel}>{label}</div>}
      {children}
    </div>
  );
}

export function EditorFieldGroup({ children }: { children: React.ReactNode }) {
  return (
    <Stack direction="row" gap={1} wrap="wrap" alignItems="flex-end">
      {children}
    </Stack>
  );
}

export function EditorField({
  label,
  grow,
  children,
}: {
  label: string;
  grow?: boolean;
  children: React.ReactElement<Record<string, unknown>>;
}) {
  const style: React.CSSProperties = grow ? { flexGrow: 1, minWidth: 0 } : {};
  return (
    <div style={style}>
      <Field label={label} noMargin>
        {children}
      </Field>
    </div>
  );
}
