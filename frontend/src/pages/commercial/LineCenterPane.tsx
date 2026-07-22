import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import {
  ApartmentOutlined,
  CloudDownloadOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";

import { HttpUtil } from "@/utils";

const { Text } = Typography;

interface LineSource {
  id: string;
  name: string;
  kind: "url" | "manual";
  urlHost?: string;
  refreshInterval: number;
  enabled: boolean;
  status: string;
  lastError?: string;
  lastSuccessAt?: string;
  groupIds: string[];
  nodeCount: number;
  healthyCount: number;
  publishedCount: number;
}

interface LineGroup {
  id: string;
  name: string;
  description: string;
  active: boolean;
  planIds: string[];
  nodeCount: number;
  healthyCount: number;
  publishedCount: number;
}

interface LineNode {
  id: string;
  fingerprint: string;
  remark: string;
  protocol: string;
  publicPort?: number;
  status: string;
  healthStatus: string;
  published: boolean;
  latencyMs: number;
  lastError?: string;
  missingSince?: string;
  sourceIds: string[];
  groupIds: string[];
}

interface PlanItem {
  plan: { id: string; name: string; active: boolean };
}

interface ImportEntry {
  index: number;
  remark?: string;
  protocol?: string;
  fingerprint?: string;
  valid: boolean;
  duplicate: boolean;
  error?: string;
}

interface ImportPreview {
  entries: ImportEntry[];
  validCount: number;
  invalidCount: number;
  duplicateCount: number;
}

interface URLSourceValues {
  name: string;
  url: string;
  refreshInterval: number;
  enabled: boolean;
  groupIds: string[];
  planIds: string[];
}

interface GroupValues {
  name: string;
  description: string;
  active: boolean;
}

interface ManualImportValues {
  name: string;
  links: string;
}

function jsonOptions(headers: Record<string, string> = {}) {
  return { headers: { "Content-Type": "application/json", ...headers } };
}

function healthColor(status: string): string {
  if (status === "healthy") return "success";
  if (["checking", "provisioning"].includes(status)) return "processing";
  if (["offline", "stale"].includes(status)) return "error";
  return "default";
}

function publicationLabel(status: string, published: boolean): string {
  if (published) return "已发布";
  if (["checking", "provisioning", "retry"].includes(status)) return "配置中";
  if (status === "stale") return "已停止";
  return "待分组";
}

function connectivityLabel(status: string): string {
  if (status === "healthy") return "探测正常";
  if (status === "offline") return "探测未通过";
  if (status === "checking") return "探测中";
  return "尚未探测";
}

function connectivityColor(status: string): string {
  if (status === "healthy") return "success";
  if (status === "checking") return "processing";
  if (status === "offline") return "warning";
  return "default";
}

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

function normalizeSource(source: LineSource): LineSource {
  return { ...source, groupIds: arrayOrEmpty(source.groupIds) };
}

function normalizeGroup(group: LineGroup): LineGroup {
  return { ...group, planIds: arrayOrEmpty(group.planIds) };
}

function normalizeNode(node: LineNode): LineNode {
  return {
    ...node,
    sourceIds: arrayOrEmpty(node.sourceIds),
    groupIds: arrayOrEmpty(node.groupIds),
  };
}

export default function LineCenterPane({ refreshToken }: { refreshToken: number }) {
  const [messageApi, contextHolder] = message.useMessage();
  const [modalApi, modalHolder] = Modal.useModal();
  const [sources, setSources] = useState<LineSource[]>([]);
  const [groups, setGroups] = useState<LineGroup[]>([]);
  const [nodes, setNodes] = useState<LineNode[]>([]);
  const [plans, setPlans] = useState<PlanItem[]>([]);
  const [selectedNodeIds, setSelectedNodeIds] = useState<string[]>([]);
  const [assignmentGroupIds, setAssignmentGroupIds] = useState<string[]>([]);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [editingSourceId, setEditingSourceId] = useState("");
  const [editingGroupId, setEditingGroupId] = useState("");
  const [urlForm] = Form.useForm<URLSourceValues>();
  const [groupForm] = Form.useForm<GroupValues>();
  const [manualForm] = Form.useForm<ManualImportValues>();
  const loadGeneration = useRef(0);

  const load = useCallback(async () => {
    const generation = ++loadGeneration.current;
    setBusy(true);
    try {
      const [sourceResult, groupResult, nodeResult, planResult] =
        await Promise.all([
          HttpUtil.get<LineSource[]>(
            "/panel/api/commercial/line-sources",
            undefined,
            { silent: true },
          ),
          HttpUtil.get<LineGroup[]>(
            "/panel/api/commercial/line-groups",
            undefined,
            { silent: true },
          ),
          HttpUtil.get<LineNode[]>(
            "/panel/api/commercial/line-nodes",
            undefined,
            { silent: true },
          ),
          HttpUtil.get<PlanItem[]>(
            "/panel/api/commercial/plans",
            undefined,
            { silent: true },
          ),
        ]);
      if (generation !== loadGeneration.current) return;
      if (sourceResult.success)
        setSources(arrayOrEmpty(sourceResult.obj).map(normalizeSource));
      if (groupResult.success)
        setGroups(arrayOrEmpty(groupResult.obj).map(normalizeGroup));
      if (nodeResult.success)
        setNodes(arrayOrEmpty(nodeResult.obj).map(normalizeNode));
      if (planResult.success) setPlans(arrayOrEmpty(planResult.obj));
    } finally {
      if (generation === loadGeneration.current) setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load();
    return () => {
      loadGeneration.current += 1;
    };
  }, [load, refreshToken]);

  const saveGroup = async () => {
    const values = await groupForm.validateFields();
    const result = await HttpUtil.post(
      "/panel/api/commercial/line-groups",
      { ...values, id: editingGroupId },
      jsonOptions(),
    );
    if (!result.success) return;
    groupForm.resetFields();
    groupForm.setFieldsValue({ active: true });
    setEditingGroupId("");
    messageApi.success(editingGroupId ? "线路组已更新" : "线路组已创建");
    await load();
  };

  const saveURLSource = async () => {
    const values = await urlForm.validateFields();
    const result = await HttpUtil.post(
      "/panel/api/commercial/line-sources",
      { ...values, id: editingSourceId },
      jsonOptions(),
    );
    if (!result.success) return;
    urlForm.resetFields();
    urlForm.setFieldsValue({ enabled: true, refreshInterval: 1800 });
    setEditingSourceId("");
    messageApi.success(editingSourceId ? "订阅来源已更新" : "订阅已导入，新节点会自动继承线路组和套餐");
    await load();
  };

  const previewManual = async () => {
    const values = await manualForm.validateFields();
    const result = await HttpUtil.post<ImportPreview>(
      "/panel/api/commercial/line-imports/preview",
      values,
      jsonOptions(),
    );
    if (result.success && result.obj) setPreview(result.obj);
  };

  const commitManual = async () => {
    const values = await manualForm.validateFields();
    const result = await HttpUtil.post(
      "/panel/api/commercial/line-imports/commit",
      values,
      jsonOptions(),
    );
    if (!result.success) return;
    setPreview(null);
    manualForm.resetFields();
    messageApi.success("有效节点已进入待分组池");
    await load();
  };

  const assignGroups = async () => {
    if (selectedNodeIds.length === 0) {
      messageApi.info("请先选择节点");
      return;
    }
    const result = await HttpUtil.put(
      "/panel/api/commercial/line-nodes/groups",
      { nodeIds: selectedNodeIds, groupIds: assignmentGroupIds },
      jsonOptions(),
    );
    if (!result.success) return;
    messageApi.success("节点分组已更新；托管线路将立即发布，连通探测仅作为参考");
    setSelectedNodeIds([]);
    await load();
  };

  const refreshSource = async (id: string) => {
    const result = await HttpUtil.post(
      `/panel/api/commercial/line-sources/${id}/refresh`,
      {},
      jsonOptions(),
    );
    if (!result.success) return;
    messageApi.success("已加入刷新队列");
    await load();
  };

  const probeNode = async (id: string) => {
    const result = await HttpUtil.post(
      `/panel/api/commercial/line-nodes/${id}/probe`,
      {},
      jsonOptions(),
    );
    if (!result.success) return;
    messageApi.success("已加入真实出站探测队列；探测结果不会停止线路发布");
    await load();
  };

  const confirmDelete = (
    title: string,
    description: string,
    action: (headers: Record<string, string>) => Promise<boolean>,
  ) => {
    let password = "";
    let twoFactorCode = "";
    modalApi.confirm({
      title,
      width: 560,
      okText: "确认删除",
      okButtonProps: { danger: true },
      content: (
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Alert type="warning" showIcon title={description} />
          <Input.Password placeholder="管理员密码（必填）" onChange={(event) => { password = event.target.value; }} />
          <Input placeholder="2FA 验证码（如已启用）" onChange={(event) => { twoFactorCode = event.target.value; }} />
        </Space>
      ),
      onOk: async () => {
        if (!password) {
          messageApi.error("请输入管理员密码");
          return Promise.reject(new Error("password required"));
        }
        const succeeded = await action({
          "X-Admin-Password": password,
          "X-Admin-2FA": twoFactorCode,
        });
        if (!succeeded) return Promise.reject(new Error("delete failed"));
      },
    });
  };

  const editSource = (source: LineSource) => {
    setEditingSourceId(source.id);
    urlForm.setFieldsValue({
      name: source.name,
      url: "",
      refreshInterval: source.refreshInterval || 1800,
      enabled: source.enabled,
      groupIds: source.groupIds,
      planIds: Array.from(
        new Set(
          source.groupIds.flatMap(
            (groupId) => groups.find((group) => group.id === groupId)?.planIds || [],
          ),
        ),
      ),
    });
  };

  const deleteSource = (source: LineSource) =>
    confirmDelete(
      "删除订阅来源",
      `将停止 ${source.name} 的刷新；只属于该来源的节点会进入 7 天保留期。`,
      async (headers) => {
        const result = await HttpUtil.delete(
          `/panel/api/commercial/line-sources/${source.id}`,
          undefined,
          jsonOptions(headers),
        );
        if (!result.success) return false;
        messageApi.success("订阅来源已删除");
        await load();
        return true;
      },
    );

  const editGroup = (group: LineGroup) => {
    setEditingGroupId(group.id);
    groupForm.setFieldsValue({ name: group.name, description: group.description, active: group.active });
  };

  const deleteGroup = (group: LineGroup) =>
    confirmDelete(
      "删除线路组",
      `将解除 ${group.name} 与节点及套餐的绑定，但不会立即删除节点。`,
      async (headers) => {
        const result = await HttpUtil.delete(
          `/panel/api/commercial/line-groups/${group.id}`,
          undefined,
          jsonOptions(headers),
        );
        if (!result.success) return false;
        messageApi.success("线路组已删除");
        await load();
        return true;
      },
    );

  const groupById = useMemo(
    () => new Map(groups.map((group) => [group.id, group])),
    [groups],
  );
  const groupOptions = useMemo(
    () => groups
      .filter((group) => group.active)
      .map((group) => ({ value: group.id, label: group.name })),
    [groups],
  );
  const planOptions = useMemo(
    () => plans.map((item) => ({
      value: item.plan.id,
      label: `${item.plan.name}${item.plan.active ? "" : "（草稿）"}`,
    })),
    [plans],
  );

  return (
    <Space orientation="vertical" size="large" style={{ width: "100%" }}>
      {contextHolder}
      {modalHolder}
      <Alert
        type="warning"
        showIcon
        title="必须在服务器防火墙和云安全组放行 TCP 20000-59999"
        description="导入协议会转换为本站 VLESS Reality 线路并立即发布；上游连通探测只用于诊断，不会再把节点移出用户订阅。若客户端延迟为 -1，优先检查这段 TCP 端口范围是否已同时放行。"
      />
      <Row gutter={[16, 16]}>
        <Col xs={12} lg={6}>
          <Card size="small"><Statistic title="来源" value={sources.length} /></Card>
        </Col>
        <Col xs={12} lg={6}>
          <Card size="small"><Statistic title="线路组" value={groups.length} /></Card>
        </Col>
        <Col xs={12} lg={6}>
          <Card size="small"><Statistic title="全部节点" value={nodes.length} /></Card>
        </Col>
        <Col xs={12} lg={6}>
          <Card size="small"><Statistic title="已发布线路" value={nodes.filter((node) => node.published).length} /></Card>
        </Col>
      </Row>
      <Tabs
        animated={false}
        items={[
          {
            key: "url",
            label: "订阅 URL 自动分配",
            children: (
              <Row gutter={[16, 16]}>
                <Col xs={24} xl={10}>
                  <Card title="添加订阅来源">
                    <Form
                      form={urlForm}
                      layout="vertical"
                      initialValues={{ refreshInterval: 1800, enabled: true, groupIds: [], planIds: [] }}
                    >
                      <Form.Item name="name" label="来源名称" rules={[{ required: true }]}><Input /></Form.Item>
                      <Form.Item name="url" label="订阅 URL" rules={[{ required: true }, { type: "url" }]}><Input.Password /></Form.Item>
                      <Form.Item name="groupIds" label="自动加入线路组" rules={[{ required: true }]}><Select mode="multiple" options={groupOptions} /></Form.Item>
                      <Form.Item name="planIds" label="同时绑定套餐"><Select mode="multiple" options={planOptions} /></Form.Item>
                      <Form.Item name="refreshInterval" label="刷新周期（秒）"><InputNumber min={300} max={86400} style={{ width: "100%" }} /></Form.Item>
                      <Form.Item name="enabled" label="自动刷新" valuePropName="checked"><Switch /></Form.Item>
                      <Space.Compact block>
                        <Button type="primary" icon={<CloudDownloadOutlined />} onClick={saveURLSource} loading={busy} block>{editingSourceId ? "验证并保存修改" : "导入并验证"}</Button>
                        {editingSourceId ? <Button onClick={() => { setEditingSourceId(""); urlForm.resetFields(); urlForm.setFieldsValue({ enabled: true, refreshInterval: 1800 }); }}>取消</Button> : null}
                      </Space.Compact>
                    </Form>
                  </Card>
                </Col>
                <Col xs={24} xl={14}>
                  <Card title="已保存来源">
                    <Table
                      rowKey="id"
                      loading={busy}
                      dataSource={sources}
                      pagination={false}
                      columns={[
                        { title: "名称", dataIndex: "name" },
                        { title: "类型", dataIndex: "kind", render: (value: string) => <Tag>{value === "url" ? "URL" : "手动"}</Tag> },
                        { title: "域名", dataIndex: "urlHost", render: (value?: string) => value || "—" },
                        { title: "已发布/全部", render: (_, row) => `${row.publishedCount || 0}/${row.nodeCount}` },
                        { title: "状态", dataIndex: "status", render: (value: string) => <Tag color={healthColor(value)}>{value}</Tag> },
                        { title: "错误", dataIndex: "lastError", ellipsis: true, render: (value?: string) => value || "—" },
                        { title: "操作", render: (_, row) => <Space size={4}>{row.kind === "url" ? <Button size="small" icon={<EditOutlined />} onClick={() => editSource(row)}>编辑</Button> : null}{row.kind === "url" ? <Button size="small" icon={<ReloadOutlined />} onClick={() => refreshSource(row.id)}>刷新</Button> : null}<Button size="small" danger icon={<DeleteOutlined />} onClick={() => deleteSource(row)}>删除</Button></Space> },
                      ]}
                    />
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: "manual",
            label: "协议链接批量导入",
            children: (
              <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
                <Card>
                  <Form form={manualForm} layout="vertical">
                    <Form.Item name="name" label="导入批次名称"><Input /></Form.Item>
                    <Form.Item name="links" label="VMess、VLESS、Trojan、Shadowsocks、Hysteria2 或 WireGuard 链接" rules={[{ required: true }]}>
                      <Input.TextArea rows={9} placeholder="每行一条协议链接" />
                    </Form.Item>
                    <Space>
                      <Button icon={<ThunderboltOutlined />} onClick={previewManual}>预览验证</Button>
                      <Button type="primary" onClick={commitManual} disabled={!preview || preview.validCount === 0}>提交有效节点</Button>
                    </Space>
                  </Form>
                </Card>
                {preview ? (
                  <Card title={`有效 ${preview.validCount} · 失败 ${preview.invalidCount} · 重复 ${preview.duplicateCount}`}>
                    <Table
                      rowKey="index"
                      dataSource={preview.entries}
                      pagination={{ pageSize: 20 }}
                      columns={[
                        { title: "#", dataIndex: "index", width: 64 },
                        { title: "备注", dataIndex: "remark", render: (value?: string) => value || "—" },
                        { title: "协议", dataIndex: "protocol", render: (value?: string) => value ? <Tag>{value}</Tag> : "—" },
                        { title: "结果", render: (_, row) => <Tag color={row.valid ? (row.duplicate ? "warning" : "success") : "error"}>{row.valid ? (row.duplicate ? "重复" : "有效") : "失败"}</Tag> },
                        { title: "原因", dataIndex: "error", render: (value?: string) => value || "—" },
                      ]}
                    />
                  </Card>
                ) : null}
              </Space>
            ),
          },
          {
            key: "groups",
            label: "分组与套餐线路",
            children: (
              <Row gutter={[16, 16]}>
                <Col xs={24} xl={8}>
                  <Card title="新建线路组">
                    <Form form={groupForm} layout="vertical" initialValues={{ active: true }}>
                      <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
                      <Form.Item name="description" label="说明"><Input.TextArea rows={3} /></Form.Item>
                      <Form.Item name="active" label="启用" valuePropName="checked"><Switch /></Form.Item>
                      <Space.Compact block>
                        <Button type="primary" icon={<PlusOutlined />} onClick={saveGroup} block>{editingGroupId ? "保存线路组" : "创建线路组"}</Button>
                        {editingGroupId ? <Button onClick={() => { setEditingGroupId(""); groupForm.resetFields(); groupForm.setFieldsValue({ active: true }); }}>取消</Button> : null}
                      </Space.Compact>
                    </Form>
                  </Card>
                </Col>
                <Col xs={24} xl={16}>
                  <Card title="线路组">
                    <Table
                      rowKey="id"
                      dataSource={groups}
                      pagination={false}
                      columns={[
                        { title: "名称", dataIndex: "name" },
                        { title: "说明", dataIndex: "description", ellipsis: true },
                        { title: "已发布/全部", render: (_, row) => `${row.publishedCount || 0}/${row.nodeCount}` },
                        { title: "套餐数", dataIndex: "planIds", render: (value: string[]) => value.length },
                        { title: "状态", dataIndex: "active", render: (value: boolean) => <Tag color={value ? "success" : "default"}>{value ? "启用" : "停用"}</Tag> },
                        { title: "操作", render: (_, row) => <Space size={4}><Button size="small" icon={<EditOutlined />} onClick={() => editGroup(row)}>编辑</Button><Button size="small" danger icon={<DeleteOutlined />} onClick={() => deleteGroup(row)}>删除</Button></Space> },
                      ]}
                    />
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: "nodes",
            label: "节点池",
            children: (
              <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
                <Card size="small">
                  <Space wrap>
                    <Select mode="multiple" style={{ minWidth: 320 }} placeholder="选择节点所属线路组" value={assignmentGroupIds} onChange={setAssignmentGroupIds} options={groupOptions} />
                    <Button type="primary" icon={<ApartmentOutlined />} onClick={assignGroups}>批量分组（{selectedNodeIds.length}）</Button>
                    <Text type="secondary">不选线路组后提交可将节点放回待分组池。</Text>
                  </Space>
                </Card>
                <Table
                  rowKey="id"
                  loading={busy}
                  dataSource={nodes}
                  scroll={{ x: 1100 }}
                  pagination={{ pageSize: 50 }}
                  rowSelection={{ selectedRowKeys: selectedNodeIds, onChange: (keys) => setSelectedNodeIds(keys.map(String)) }}
                  columns={[
                    { title: "节点", dataIndex: "remark", width: 220 },
                    { title: "协议", dataIndex: "protocol", render: (value: string) => <Tag>{value}</Tag> },
                    { title: "公网端口", dataIndex: "publicPort", render: (value?: number) => value || "待分配" },
                    { title: "延迟", dataIndex: "latencyMs", render: (value: number) => value > 0 ? `${value} ms` : "—" },
                    { title: "线路组", dataIndex: "groupIds", render: (values: string[]) => arrayOrEmpty(values).map((id) => <Tag key={id}>{groupById.get(id)?.name || id.slice(0, 8)}</Tag>) },
                    { title: "发布状态", dataIndex: "status", render: (value: string, row) => <Tag color={row.published ? "success" : healthColor(value)}>{publicationLabel(value, row.published)}</Tag> },
                    { title: "连通参考", dataIndex: "healthStatus", render: (value: string) => <Tag color={connectivityColor(value)}>{connectivityLabel(value)}</Tag> },
                    { title: "探测信息", dataIndex: "lastError", ellipsis: true, render: (value?: string) => value || "—" },
                    { title: "操作", fixed: "right", width: 120, render: (_, row) => <Button size="small" icon={<ThunderboltOutlined />} onClick={() => probeNode(row.id)}>重新探测</Button> },
                  ]}
                />
              </Space>
            ),
          },
        ]}
      />
    </Space>
  );
}
