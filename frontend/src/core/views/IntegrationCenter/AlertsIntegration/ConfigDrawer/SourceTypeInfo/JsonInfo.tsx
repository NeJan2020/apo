/**
 * Copyright 2025 CloudDetail
 * SPDX-License-Identifier: Apache-2.0
 */

import { Typography, Table } from 'antd'
import { useTranslation } from 'react-i18next'
import Title from 'antd/es/typography/Title'
import Text from 'antd/es/typography/Text'
import CopyPre from 'src/core/components/CopyPre'

const JsonInfo = () => {
  const { t } = useTranslation('core/alertsIntegration')
  const columns1 = [
    { dataIndex: 'title', title: t('jsonInfo.table.fieldName'), width: 150 },
    { dataIndex: 'meaning', title: t('jsonInfo.table.meaning'), width: 150 },
    { dataIndex: 'type', title: t('jsonInfo.table.type'), width: 150 },
    { dataIndex: 'description', title: t('jsonInfo.table.description') },
  ]

  const data1 = [
    {
      title: 'sourceId',
      meaning: t('jsonInfo.fields.sourceId.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.sourceId.description'),
    },
  ]

  const data2 = [
    {
      title: 'name',
      meaning: t('jsonInfo.fields.name.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.name.description'),
    },
    {
      title: 'status',
      meaning: t('jsonInfo.fields.status.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.status.description'),
    },
    {
      title: 'severity',
      meaning: t('jsonInfo.fields.severity.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.severity.description'),
    },
    {
      title: 'detail',
      meaning: t('jsonInfo.fields.detail.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.detail.description'),
    },
    {
      title: 'alertId',
      meaning: t('jsonInfo.fields.alertId.meaning'),
      type: 'string',
      description: t('jsonInfo.fields.alertId.description'),
    },
    {
      title: 'tags',
      meaning: t('jsonInfo.fields.tags.meaning'),
      type: 'map[string]string',
      description: t('jsonInfo.fields.tags.description'),
    },
    {
      title: 'startTime',
      meaning: t('jsonInfo.fields.startTime.meaning'),
      type: 'int64',
      description: t('jsonInfo.fields.startTime.description'),
    },
    {
      title: 'updateTime',
      meaning: t('jsonInfo.fields.updateTime.meaning'),
      type: 'int64',
      description: t('jsonInfo.fields.updateTime.description'),
    },
    {
      title: 'endTime',
      meaning: t('jsonInfo.fields.endTime.meaning'),
      type: 'int64',
      description: t('jsonInfo.fields.endTime.description'),
    },
  ]

  const code1 = `{
    "name": "Zabbix-Trigger-1",
    "status": "trigger",
    "detail": "Zabbix-Trigger-1",
    "alertId": "1234567890",
    "tags": {
        "tag1": "value1",
        "tag2": "value2"
    },
    "startTime": "2025-01-21T15:04:05+00:00",
    "updateTime": 1737514800000,
    "endTime": 1737514800000,
    "severity": "error",
    "status": "firing"
}`

  const code2 = `{
    "code": "B1319",
    "message":"${t('jsonInfo.code2Error')}"
}`

  return (
    <Typography>
      <Title level={4}>{t('jsonInfo.title')}</Title>
      <Text>{t('jsonInfo.description')}</Text>

      <Title level={5}>{t('jsonInfo.interface.title')}</Title>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.interface.method')}</Text>
        <Typography>POST</Typography>
      </Typography>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.interface.headers')}</Text>
        <Typography>Content-Type: application/json</Typography>
      </Typography>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.interface.params')}</Text>
        <Table columns={columns1} dataSource={data1} pagination={false} />
      </Typography>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.interface.body')}</Text>
        <Table columns={columns1} dataSource={data2} pagination={false} />
      </Typography>

      <Title level={5}>{t('jsonInfo.example.title')}</Title>
      <Typography className="mb-2">
        <CopyPre code={code1} />
      </Typography>

      <Title level={5}>{t('jsonInfo.response.title')}</Title>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.response.success')}</Text>
        <Typography>200 "ok"</Typography>
      </Typography>
      <Typography className="mb-2">
        <Text strong>{t('jsonInfo.response.failure')}</Text>
        <CopyPre code={code2} />
      </Typography>
    </Typography>
  )
}

export default JsonInfo
