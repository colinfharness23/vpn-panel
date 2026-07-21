import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Col,
  ConfigProvider,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  Layout,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
  message,
} from "antd";
import type { TableProps, UploadFile } from "antd";
import {
  AuditOutlined,
  ApartmentOutlined,
  BankOutlined,
  CustomerServiceOutlined,
  DeleteOutlined,
  EditOutlined,
  GiftOutlined,
  NotificationOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SendOutlined,
  SettingOutlined,
  ShoppingCartOutlined,
  TeamOutlined,
  UploadOutlined,
  UserAddOutlined,
} from "@ant-design/icons";
import dayjs, { type Dayjs } from "dayjs";
import { useLocation, useNavigate } from "react-router-dom";

import { useTheme } from "@/hooks/useTheme";
import AppSidebar from "@/layouts/AppSidebar";
import { HttpUtil } from "@/utils";
import { setMessageInstance } from "@/utils/messageBus";
import InvitationSettingsPane from "./InvitationSettingsPane";
import LineCenterPane from "./LineCenterPane";
import PaymentSettingsView, {
  type PaymentSettingsFormValues,
} from "./PaymentSettingsView";
import { useInvitationSettings } from "./useInvitationSettings";
import "./CommercialPage.css";

const { Text, Paragraph } = Typography;
const GB = 1024 ** 3;
const locales = [
  "zh-CN",
  "zh-TW",
  "en-US",
  "ja-JP",
  "ru-RU",
  "vi-VN",
  "es-ES",
  "id-ID",
  "uk-UA",
  "tr-TR",
  "pt-BR",
  "ar-EG",
  "fa-IR",
];
const commercialTabKeys = [
  "overview",
  "lines",
  "customers",
  "plans",
  "orders",
  "content",
  "tickets",
  "marketing",
  "settings",
  "roles",
  "audit",
];

interface Overview {
  customers: number;
  activeEntitlements: number;
  pendingOrders: number;
  openTickets: number;
  manualJobs: number;
  revenueFen: number;
  orderStatus: Record<string, number>;
}
interface Entitlement {
  id: string;
  planId: string;
  status: string;
  trafficQuota: number;
  trafficUsed: number;
  deviceLimit: number;
  trafficMultiplierPermille: number;
  uploadLimitMbps: number;
  downloadLimitMbps: number;
  startsAt: string;
  expiresAt?: string;
}
interface CustomerSubscription {
  entitlement: Entitlement;
  plan: Plan;
}
interface Customer {
  id: string;
  email: string;
  displayName: string;
  status: string;
  balanceFen: number;
  createdAt: string;
  systemAdmin: boolean;
  subscription?: CustomerSubscription;
}
interface Order {
  id: string;
  outTradeNo: string;
  customerId: string;
  orderKind?: "purchase" | "renewal" | "upgrade";
  entitlementId?: string;
  status: string;
  payableFen: number;
  paidFen: number;
  balancePaidFen: number;
  failureReason: string;
  createdAt: string;
}
interface Plan {
  id: string;
  name: string;
  slug: string;
  description: string;
  trafficBytes: number;
  deviceLimit: number;
  trafficMultiplierPermille: number;
  uploadLimitMbps: number;
  downloadLimitMbps: number;
  residentialRelayEnabled: boolean;
  residentialRelayLimit: number;
  resetCycle: string;
  nodeGroup: string;
  capacity: number;
  active: boolean;
  visibility: string;
  renewable: boolean;
  upgradable: boolean;
  sortOrder: number;
  provisionInboundIds: string;
  displayBenefits: PlanDisplayBenefits;
}
interface PlanDisplayBenefits {
  globalCoverage: string;
  standardNodes: string;
  advancedNodes: string;
  premiumRoutes: string;
  residentialIpSale: string;
  socialMedia: string;
  crossBorderWork: string;
  liveStreaming: string;
  uploadOptimization: string;
  peakPriority: string;
  failover: string;
  support: string;
}
interface PlanPrice {
  id: string;
  planId: string;
  billingPeriod: string;
  months: number;
  amountFen: number;
  active: boolean;
}
interface PlanItem {
  plan: Plan;
  prices: PlanPrice[];
  lineGroupIds: string[];
}
interface LineGroupSummary {
  id: string;
  name: string;
  active: boolean;
}
interface Notice {
  id: string;
  slug: string;
  level: string;
  titleI18n: string;
  contentI18n: string;
  published: boolean;
  updatedAt: string;
}
interface Article {
  id: string;
  slug: string;
  category: string;
  titleI18n: string;
  contentI18n: string;
  published: boolean;
  sortOrder: number;
  updatedAt: string;
}
interface Application {
  id: string;
  slug: string;
  name: string;
  platform: string;
  description: string;
  packageFileName?: string;
  packageSize?: number;
  packageSha256?: string;
  downloadUrl?: string;
  active: boolean;
  sortOrder: number;
  updatedAt: string;
}
interface Ticket {
  id: string;
  customerId: string;
  entitlementId?: string;
  planId?: string;
  planName?: string;
  subject: string;
  status: string;
  priority: string;
  updatedAt: string;
}
interface TicketMessage {
  id: string;
  senderType: string;
  body: string;
  createdAt: string;
}
interface Coupon {
  id: string;
  code: string;
  kind: string;
  value: number;
  minimumFen: number;
  maxRedemptions: number;
  redeemedCount: number;
  startsAt?: string;
  expiresAt?: string;
  active: boolean;
}
interface GiftCard {
  id: string;
  displayCode: string;
  valueFen: number;
  status: string;
  createdAt: string;
}
interface Commission {
  id: string;
  inviterId: string;
  inviteeId: string;
  amountFen: number;
  status: string;
}
interface AuditLog {
  id: string;
  actorUserId: number;
  actorRole: string;
  action: string;
  targetType: string;
  targetId: string;
  createdAt: string;
}
interface AdminRole {
  userId: number;
  username: string;
  role: string;
}
interface Paginated<T> {
  items: T[];
  total: number;
}

type PlanFormValues = Omit<
  Plan,
  "id" | "trafficBytes" | "trafficMultiplierPermille" | "provisionInboundIds"
> & {
  trafficGB: number;
  trafficMultiplier: number;
  provisionInboundIds: string;
  lineGroupIds: string[];
};

const emptyPlanDisplayBenefits = (): PlanDisplayBenefits => ({
  globalCoverage: "",
  standardNodes: "",
  advancedNodes: "",
  premiumRoutes: "",
  residentialIpSale: "",
  socialMedia: "",
  crossBorderWork: "",
  liveStreaming: "",
  uploadOptimization: "",
  peakPriority: "",
  failover: "",
  support: "",
});

const planDisplayBenefitFields: Array<{
  key: keyof PlanDisplayBenefits;
  label: string;
}> = [
  { key: "globalCoverage", label: "覆盖100+国家和地区" },
  { key: "standardNodes", label: "全球标准节点" },
  { key: "advancedNodes", label: "高级专线节点" },
  { key: "premiumRoutes", label: "IEPL/IPLC 高速线路" },
  { key: "residentialIpSale", label: "提供或销售住宅 IP" },
  { key: "socialMedia", label: "海外社交媒体" },
  { key: "crossBorderWork", label: "跨境电商与远程办公" },
  { key: "liveStreaming", label: "海外直播" },
  { key: "uploadOptimization", label: "直播上行优化" },
  { key: "peakPriority", label: "高峰期线路优先级" },
  { key: "failover", label: "线路故障切换" },
  { key: "support", label: "客服响应" },
];
type CustomerFormValues = { status: string; balanceYuan: number };
type CustomerCreateFormValues = {
  email: string;
  password: string;
  displayName: string;
  locale: string;
  status: string;
};
type SubscriptionFormValues = {
  planId: string;
  expiresAt?: Dayjs;
  trafficGB: number;
  deviceLimit: number;
  trafficMultiplier: number;
  uploadLimitMbps: number;
  downloadLimitMbps: number;
  resetTraffic: boolean;
  password: string;
  twoFactorCode: string;
};
type LocalizedFormValues = {
  slug: string;
  level?: string;
  category?: string;
  published: boolean;
  sortOrder?: number;
  titles: Record<string, string>;
  contents: Record<string, string>;
};
type ApplicationFormValues = Pick<
  Application,
  "slug" | "name" | "platform" | "description" | "active" | "sortOrder"
>;
type CouponFormValues = {
  code: string;
  kind: string;
  value: number;
  minimumYuan: number;
  maxRedemptions: number;
  startsAt?: string;
  expiresAt?: string;
  active: boolean;
};

function money(fen: number): string {
  return `¥ ${(fen / 100).toFixed(2)}`;
}
function date(value?: string): string {
  return value
    ? new Intl.DateTimeFormat("zh-CN", {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(value))
    : "—";
}
function jsonOptions(headers: Record<string, string> = {}) {
  return { headers: { "Content-Type": "application/json", ...headers } };
}
function fileSize(bytes?: number): string {
  if (!bytes || bytes < 1) return "—";
  const units = ["B", "KB", "MB", "GB"];
  const unit = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / 1024 ** unit).toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}
function statusColor(status: string): string {
  if (
    [
      "active",
      "paid",
      "completed",
      "confirmed",
      "settled",
      "open",
      "published",
    ].includes(status)
  )
    return "success";
  if (["pending", "provisioning", "retry"].includes(status))
    return "processing";
  if (["suspended", "manual", "provisioning_failed"].includes(status))
    return "warning";
  return "default";
}

export function CustomerBulkDeleteButton({
  selectedCount,
  onDelete,
  onEmpty,
}: {
  selectedCount: number;
  onDelete: () => void;
  onEmpty: () => void;
}) {
  const handleClick = () => {
    if (selectedCount === 0) {
      onEmpty();
      return;
    }
    onDelete();
  };
  return (
    <Button
      type="primary"
      danger
      icon={<DeleteOutlined />}
      aria-label={
        selectedCount > 0
          ? `批量删除已选择的 ${selectedCount} 个客户`
          : "批量删除客户"
      }
      onClick={handleClick}
    >
      批量删除{selectedCount > 0 ? `（${selectedCount}）` : ""}
    </Button>
  );
}
function parseI18n(raw?: string): Record<string, string> {
  try {
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}
function parseInboundIDs(raw?: string): string {
  try {
    const ids = JSON.parse(raw || "[]");
    return Array.isArray(ids) ? ids.join(",") : "";
  } catch {
    return "";
  }
}

export default function CommercialPage() {
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const { hash } = useLocation();
  const navigate = useNavigate();
  const [messageApi, messageContextHolder] = message.useMessage();
  const [modalApi, modalContextHolder] = Modal.useModal();
  const requestedTab = hash.replace(/^#/, "");
  const activeTab = commercialTabKeys.includes(requestedTab)
    ? requestedTab
    : "overview";
  const [loading, setLoading] = useState(false);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [customerTotal, setCustomerTotal] = useState(0);
  const [customerPage, setCustomerPage] = useState(1);
  const [customerSearch, setCustomerSearch] = useState("");
  const [customerStatus, setCustomerStatus] = useState("");
  const [orders, setOrders] = useState<Order[]>([]);
  const [plans, setPlans] = useState<PlanItem[]>([]);
  const [lineGroups, setLineGroups] = useState<LineGroupSummary[]>([]);
  const [lineReloadToken, setLineReloadToken] = useState(0);
  const [notices, setNotices] = useState<Notice[]>([]);
  const [articles, setArticles] = useState<Article[]>([]);
  const [applications, setApplications] = useState<Application[]>([]);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [coupons, setCoupons] = useState<Coupon[]>([]);
  const [giftCards, setGiftCards] = useState<GiftCard[]>([]);
  const [commissions, setCommissions] = useState<Commission[]>([]);
  const [settings, setSettings] = useState<Record<string, string | boolean>>(
    {},
  );
  const [audit, setAudit] = useState<AuditLog[]>([]);
  const [adminRoles, setAdminRoles] = useState<AdminRole[]>([]);

  const [editingCustomer, setEditingCustomer] = useState<Customer | null>(null);
  const [creatingCustomer, setCreatingCustomer] = useState(false);
  const [subscriptionCustomer, setSubscriptionCustomer] =
    useState<Customer | null>(null);
  const [selectedCustomerIDs, setSelectedCustomerIDs] = useState<string[]>([]);
  const [editingPlan, setEditingPlan] = useState<PlanItem | null | undefined>(
    undefined,
  );
  const [planPrices, setPlanPrices] = useState<PlanPrice[]>([]);
  const [contentEditor, setContentEditor] = useState<{
    kind: "notice" | "article";
    row?: Notice | Article;
  } | null>(null);
  const [editingApplication, setEditingApplication] = useState<
    Application | null | undefined
  >(undefined);
  const [applicationPackageFiles, setApplicationPackageFiles] = useState<
    UploadFile[]
  >([]);
  const [editingCoupon, setEditingCoupon] = useState<Coupon | null | undefined>(
    undefined,
  );
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [ticketMessages, setTicketMessages] = useState<TicketMessage[]>([]);

  const [customerForm] = Form.useForm<CustomerFormValues>();
  const [customerCreateForm] = Form.useForm<CustomerCreateFormValues>();
  const [subscriptionForm] = Form.useForm<SubscriptionFormValues>();
  const [planForm] = Form.useForm<PlanFormValues>();
  const [contentForm] = Form.useForm<LocalizedFormValues>();
  const [applicationForm] = Form.useForm<ApplicationFormValues>();
  const [couponForm] = Form.useForm<CouponFormValues>();
  const [replyForm] = Form.useForm<{ body: string; status: string }>();

  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const loadTab = useCallback(
    async (key: string) => {
      // LineCenterPane owns its initial load. Incrementing its refresh token
      // here caused a second concurrent mount-time request and an avoidable
      // nested Tabs re-render while the first result was committing.
      if (key === "lines") return;
      setLoading(true);
      try {
        if (key === "overview") {
          const result = await HttpUtil.get<Overview>(
            "/panel/api/commercial/overview",
            undefined,
            { silent: true },
          );
          if (result.success && result.obj) setOverview(result.obj);
        } else if (key === "customers") {
          const [customerResult, planResult] = await Promise.all([
            HttpUtil.get<Paginated<Customer>>(
              "/panel/api/commercial/customers",
              {
                page: customerPage,
                pageSize: 100,
                search: customerSearch || undefined,
                status: customerStatus || undefined,
              },
              { silent: true },
            ),
            HttpUtil.get<PlanItem[]>("/panel/api/commercial/plans", undefined, {
              silent: true,
            }),
          ]);
          if (customerResult.success && customerResult.obj) {
            setCustomers(customerResult.obj.items);
            setCustomerTotal(customerResult.obj.total);
          }
          if (planResult.success && planResult.obj) setPlans(planResult.obj);
        } else if (key === "plans") {
          const [planResult, groupResult] = await Promise.all([
            HttpUtil.get<PlanItem[]>(
              "/panel/api/commercial/plans",
              undefined,
              { silent: true },
            ),
            HttpUtil.get<LineGroupSummary[]>(
              "/panel/api/commercial/line-groups",
              undefined,
              { silent: true },
            ),
          ]);
          if (planResult.success && planResult.obj) setPlans(planResult.obj);
          if (groupResult.success && groupResult.obj)
            setLineGroups(groupResult.obj);
        } else if (key === "orders") {
          const result = await HttpUtil.get<Paginated<Order>>(
            "/panel/api/commercial/orders",
            { page: 1, pageSize: 100 },
            { silent: true },
          );
          if (result.success && result.obj) setOrders(result.obj.items);
        } else if (key === "content") {
          const [noticeResult, articleResult, applicationResult] =
            await Promise.all([
              HttpUtil.get<Notice[]>(
                "/panel/api/commercial/notices",
                undefined,
                { silent: true },
              ),
              HttpUtil.get<Article[]>(
                "/panel/api/commercial/articles",
                undefined,
                { silent: true },
              ),
              HttpUtil.get<Application[]>(
                "/panel/api/commercial/applications",
                undefined,
                { silent: true },
              ),
            ]);
          if (noticeResult.success && noticeResult.obj)
            setNotices(noticeResult.obj);
          if (articleResult.success && articleResult.obj)
            setArticles(articleResult.obj);
          if (applicationResult.success && applicationResult.obj)
            setApplications(applicationResult.obj);
        } else if (key === "tickets") {
          const result = await HttpUtil.get<Ticket[]>(
            "/panel/api/commercial/tickets",
            undefined,
            { silent: true },
          );
          if (result.success && result.obj) setTickets(result.obj);
        } else if (key === "marketing") {
          const [couponResult, giftResult, commissionResult] =
            await Promise.all([
              HttpUtil.get<Coupon[]>(
                "/panel/api/commercial/coupons",
                undefined,
                { silent: true },
              ),
              HttpUtil.get<GiftCard[]>(
                "/panel/api/commercial/gift-cards",
                undefined,
                { silent: true },
              ),
              HttpUtil.get<Commission[]>(
                "/panel/api/commercial/commissions",
                undefined,
                { silent: true },
              ),
            ]);
          if (couponResult.success && couponResult.obj)
            setCoupons(couponResult.obj);
          if (giftResult.success && giftResult.obj)
            setGiftCards(giftResult.obj);
          if (commissionResult.success && commissionResult.obj)
            setCommissions(commissionResult.obj);
        } else if (key === "settings") {
          const result = await HttpUtil.get<Record<string, string | boolean>>(
            "/panel/api/commercial/settings",
            undefined,
            { silent: true },
          );
          if (result.success && result.obj) setSettings(result.obj);
        } else if (key === "audit") {
          const result = await HttpUtil.get<AuditLog[]>(
            "/panel/api/commercial/audit",
            { limit: 200 },
            { silent: true },
          );
          if (result.success && result.obj) setAudit(result.obj);
        } else if (key === "roles") {
          const result = await HttpUtil.get<AdminRole[]>(
            "/panel/api/commercial/roles",
            undefined,
            { silent: true },
          );
          if (result.success && result.obj) setAdminRoles(result.obj);
        }
      } finally {
        setLoading(false);
      }
    },
    [customerPage, customerSearch, customerStatus],
  );

  useEffect(() => {
    loadTab(activeTab);
  }, [activeTab, loadTab]);

  const setActiveTab = useCallback(
    (key: string) => {
      navigate(`/commercial#${key}`, { replace: true });
    },
    [navigate],
  );

  const refreshActiveTab = useCallback(() => {
    if (activeTab === "lines") {
      setLineReloadToken((current) => current + 1);
      return;
    }
    void loadTab(activeTab);
  }, [activeTab, loadTab]);

  const openCustomer = (row: Customer) => {
    setEditingCustomer(row);
    customerForm.setFieldsValue({
      status: row.status,
      balanceYuan: row.balanceFen / 100,
    });
  };
  const saveCustomer = async () => {
    if (!editingCustomer) return;
    const values = await customerForm.validateFields();
    const result = await HttpUtil.patch(
      `/panel/api/commercial/customers/${editingCustomer.id}`,
      {
        status: values.status,
        balanceFen: Math.round(values.balanceYuan * 100),
      },
      jsonOptions(),
    );
    if (result.success) {
      messageApi.success("客户资料已更新");
      setEditingCustomer(null);
      loadTab("customers");
    }
  };

  const openCreateCustomer = () => {
    customerCreateForm.setFieldsValue({
      email: "",
      password: "",
      displayName: "",
      locale: "zh-CN",
      status: "active",
    });
    setCreatingCustomer(true);
  };
  const saveCreatedCustomer = async () => {
    const values = await customerCreateForm.validateFields();
    const result = await HttpUtil.post<Customer>(
      "/panel/api/commercial/customers",
      values,
      jsonOptions(),
    );
    if (result.success) {
      messageApi.success("客户已创建，可立即登录前台");
      setCreatingCustomer(false);
      loadTab("customers");
    }
  };

  const openSubscription = (row: Customer) => {
    const current = row.subscription;
    const plan = current?.plan || plans[0]?.plan;
    setSubscriptionCustomer(row);
    subscriptionForm.setFieldsValue({
      planId: plan?.id || "",
      expiresAt: current?.entitlement.expiresAt
        ? dayjs(current.entitlement.expiresAt)
        : dayjs().add(1, "month"),
      trafficGB:
        (current?.entitlement.trafficQuota || plan?.trafficBytes || 0) / GB,
      deviceLimit: current?.entitlement.deviceLimit ?? plan?.deviceLimit ?? 0,
      trafficMultiplier:
        (current?.entitlement.trafficMultiplierPermille ??
          plan?.trafficMultiplierPermille ??
          1000) / 1000,
      uploadLimitMbps:
        current?.entitlement.uploadLimitMbps ?? plan?.uploadLimitMbps ?? 0,
      downloadLimitMbps:
        current?.entitlement.downloadLimitMbps ?? plan?.downloadLimitMbps ?? 0,
      resetTraffic: false,
      password: "",
      twoFactorCode: "",
    });
  };
  const selectSubscriptionPlan = (planID: string) => {
    const plan = plans.find((item) => item.plan.id === planID)?.plan;
    if (!plan) return;
    subscriptionForm.setFieldsValue({
      planId: planID,
      trafficGB: plan.trafficBytes / GB,
      deviceLimit: plan.deviceLimit,
      trafficMultiplier: (plan.trafficMultiplierPermille || 1000) / 1000,
      uploadLimitMbps: plan.uploadLimitMbps || 0,
      downloadLimitMbps: plan.downloadLimitMbps || 0,
    });
  };
  const saveSubscription = async () => {
    if (!subscriptionCustomer) return;
    const values = await subscriptionForm.validateFields();
    const result = await HttpUtil.put<CustomerSubscription>(
      `/panel/api/commercial/customers/${subscriptionCustomer.id}/subscription`,
      {
        planId: values.planId,
        expiresAt: values.expiresAt?.toISOString() || "",
        trafficQuota: Math.round(values.trafficGB * GB),
        deviceLimit: values.deviceLimit,
        trafficMultiplierPermille: Math.round(values.trafficMultiplier * 1000),
        uploadLimitMbps: values.uploadLimitMbps,
        downloadLimitMbps: values.downloadLimitMbps,
        resetTraffic: values.resetTraffic,
      },
      jsonOptions({
        "X-Admin-Password": values.password,
        "X-Admin-2FA": values.twoFactorCode || "",
      }),
    );
    if (result.success) {
      messageApi.success(
        subscriptionCustomer.subscription
          ? "客户订阅已更新"
          : "客户订阅已手动开通",
      );
      setSubscriptionCustomer(null);
      loadTab("customers");
    }
  };

  const confirmSensitiveDelete = (
    title: string,
    description: string,
    action: (headers: Record<string, string>) => Promise<boolean>,
  ) => {
    let password = "";
    let twoFactorCode = "";
    modalApi.confirm({
      title,
      width: 560,
      okText: "确认永久删除",
      okButtonProps: { danger: true },
      content: (
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <Alert type="warning" showIcon title={description} />
          <Input.Password
            placeholder="管理员密码（必填）"
            onChange={(event) => {
              password = event.target.value;
            }}
          />
          <Input
            placeholder="2FA 验证码（如已启用）"
            onChange={(event) => {
              twoFactorCode = event.target.value;
            }}
          />
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
  const deleteSubscription = (row: Customer) =>
    confirmSensitiveDelete(
      "删除客户订阅",
      `将撤销 ${row.email} 的订阅链接，并从所有面板入站中删除对应客户端。客户账号与订单仍保留。`,
      async (headers) => {
        const result = await HttpUtil.delete(
          `/panel/api/commercial/customers/${row.id}/subscription`,
          undefined,
          jsonOptions(headers),
        );
        if (!result.success) return false;
        messageApi.success("客户订阅已彻底删除");
        setSubscriptionCustomer(null);
        await loadTab("customers");
        return true;
      },
    );
  const deleteCustomers = (ids: string[]) => {
    const uniqueIDs = [...new Set(ids.filter(Boolean))];
    if (uniqueIDs.length === 0) {
      messageApi.info("请先勾选需要删除的客户");
      return;
    }
    if (uniqueIDs.length > 500) {
      messageApi.error("每次最多批量删除 500 个客户");
      return;
    }
    return confirmSensitiveDelete(
      uniqueIDs.length > 1
        ? `批量删除 ${uniqueIDs.length} 个客户`
        : "永久删除客户",
      "此操作会清理账号、会话、验证码、订阅、面板客户端、订单、付款关联、工单和营销关联，且无法恢复。",
      async (headers) => {
        const result = await HttpUtil.delete<{
          deleted: string[];
          failed: Record<string, string>;
        }>(
          "/panel/api/commercial/customers",
          { ids: uniqueIDs },
          jsonOptions(headers),
        );
        if (!result.success || !result.obj) return false;
        const failedIDs = Object.keys(result.obj.failed || {});
        const failedCount = failedIDs.length;
        if (failedCount > 0)
          messageApi.warning(
            `已删除 ${result.obj.deleted.length} 个，${failedCount} 个未删除`,
          );
        else
          messageApi.success(`已彻底删除 ${result.obj.deleted.length} 个客户`);
        setSelectedCustomerIDs(failedIDs);
        await loadTab("customers");
        return true;
      },
    );
  };

  const openPlan = (item?: PlanItem) => {
    setEditingPlan(item || null);
    setPlanPrices(
      item?.prices.map((price) => ({ ...price })) || [
        {
          id: "",
          planId: "",
          billingPeriod: "monthly",
          months: 1,
          amountFen: 0,
          active: true,
        },
      ],
    );
    planForm.setFieldsValue(
      item
        ? {
            ...item.plan,
            displayBenefits:
              item.plan.displayBenefits || emptyPlanDisplayBenefits(),
            trafficGB: item.plan.trafficBytes / GB,
            trafficMultiplier:
              (item.plan.trafficMultiplierPermille || 1000) / 1000,
            residentialRelayEnabled: item.plan.residentialRelayEnabled ?? false,
            residentialRelayLimit: item.plan.residentialRelayLimit || 0,
            provisionInboundIds: parseInboundIDs(item.plan.provisionInboundIds),
            lineGroupIds: item.lineGroupIds || [],
          }
        : {
            slug: "",
            name: "",
            description: "",
            trafficGB: 100,
            deviceLimit: 3,
            trafficMultiplier: 1,
            uploadLimitMbps: 0,
            downloadLimitMbps: 0,
            residentialRelayEnabled: false,
            residentialRelayLimit: 0,
            resetCycle: "monthly",
            nodeGroup: "default",
            capacity: 0,
            visibility: "public",
            renewable: true,
            upgradable: true,
            active: false,
            sortOrder: 0,
            provisionInboundIds: "",
            lineGroupIds: [],
            displayBenefits: emptyPlanDisplayBenefits(),
          },
    );
  };
  const savePlan = async () => {
    const values = await planForm.validateFields();
    if (
      planPrices.length === 0 ||
      planPrices.some(
        (price) =>
          price.amountFen <= 0 ||
          !price.billingPeriod ||
          (price.billingPeriod !== "one_time" && price.months <= 0),
      )
    ) {
      messageApi.error("请至少配置一个金额和周期均有效的价格");
      return;
    }
    if (
      new Set(
        planPrices
          .filter((price) => price.active)
          .map((price) => price.billingPeriod),
      ).size !== planPrices.filter((price) => price.active).length
    ) {
      messageApi.error("同一种计费周期只能启用一个价格");
      return;
    }
    const inboundIDs = values.provisionInboundIds
      .split(",")
      .map((value) => Number(value.trim()))
      .filter((value) => Number.isInteger(value) && value > 0);
    const desiredActive = values.active;
    if (
      desiredActive &&
      inboundIDs.length === 0 &&
      values.lineGroupIds.length === 0
    ) {
      messageApi.error("套餐上架前必须绑定至少一个线路组或高级入站");
      return;
    }
    if (desiredActive && !planPrices.some((price) => price.active)) {
      messageApi.error("套餐上架前必须启用至少一个价格");
      return;
    }
    const { trafficMultiplier, ...planValues } = values;
    const planPayload = {
      ...planValues,
      id: editingPlan?.plan.id || "",
      trafficBytes: Math.round(values.trafficGB * GB),
      trafficMultiplierPermille: Math.round(trafficMultiplier * 1000),
      provisionInboundIds: [...new Set(inboundIDs)],
      active: editingPlan ? desiredActive : false,
    };
    const result = await HttpUtil.post<Plan>(
      "/panel/api/commercial/plans",
      planPayload,
      jsonOptions(),
    );
    if (!result.success || !result.obj) return;
    for (const removed of editingPlan?.prices.filter(
      (price) => !planPrices.some((current) => current.id === price.id),
    ) || []) {
      const disabled = await HttpUtil.post(
        "/panel/api/commercial/plan-prices",
        { ...removed, planId: result.obj.id, active: false },
        jsonOptions(),
      );
      if (!disabled.success) return;
    }
    for (const price of planPrices) {
      const saved = await HttpUtil.post(
        "/panel/api/commercial/plan-prices",
        { ...price, planId: result.obj.id },
        jsonOptions(),
      );
      if (!saved.success) return;
    }
    if (desiredActive) {
      const published = await HttpUtil.post<Plan>(
        "/panel/api/commercial/plans",
        { ...planPayload, id: result.obj.id, active: true },
        jsonOptions(),
      );
      if (!published.success) return;
    }
    messageApi.success(editingPlan ? "套餐与价格已更新" : "套餐已创建");
    setEditingPlan(undefined);
    loadTab("plans");
  };

  const openContent = (kind: "notice" | "article", row?: Notice | Article) => {
    setContentEditor({ kind, row });
    contentForm.setFieldsValue({
      slug: row?.slug || "",
      level:
        kind === "notice"
          ? (row as Notice | undefined)?.level || "info"
          : undefined,
      category:
        kind === "article"
          ? (row as Article | undefined)?.category || ""
          : undefined,
      published: row?.published ?? false,
      sortOrder:
        kind === "article"
          ? (row as Article | undefined)?.sortOrder || 0
          : undefined,
      titles: parseI18n(row?.titleI18n),
      contents: parseI18n(row?.contentI18n),
    });
  };
  const saveContent = async () => {
    if (!contentEditor) return;
    const values = await contentForm.validateFields();
    const endpoint = contentEditor.kind === "notice" ? "notices" : "articles";
    const result = await HttpUtil.post(
      `/panel/api/commercial/${endpoint}`,
      {
        ...values,
        id: contentEditor.row?.id || "",
        titleI18n: JSON.stringify(values.titles || {}),
        contentI18n: JSON.stringify(values.contents || {}),
      },
      jsonOptions(),
    );
    if (result.success) {
      messageApi.success(contentEditor.row ? "内容已更新" : "内容已创建");
      setContentEditor(null);
      loadTab("content");
    }
  };

  const openApplication = (row?: Application) => {
    setEditingApplication(row || null);
    setApplicationPackageFiles([]);
    applicationForm.setFieldsValue(
      row || {
        slug: "",
        name: "",
        platform: "Windows",
        description: "",
        active: true,
        sortOrder: 0,
      },
    );
  };
  const saveApplication = async () => {
    const values = await applicationForm.validateFields();
    const packageFile = applicationPackageFiles[0]?.originFileObj;
    if (!editingApplication?.packageFileName && !packageFile) {
      messageApi.error("请选择要提供给用户下载的安装包");
      return;
    }
    const result = await HttpUtil.post<Application>(
      "/panel/api/commercial/applications",
      { ...values, id: editingApplication?.id || "" },
      jsonOptions(),
    );
    if (!result.success || !result.obj) return;
    if (packageFile) {
      const formData = new FormData();
      formData.append("package", packageFile, packageFile.name);
      const uploadResult = await HttpUtil.post<Application>(
        `/panel/api/commercial/applications/${result.obj.id}/package`,
        formData,
      );
      if (!uploadResult.success) {
        setEditingApplication(result.obj);
        return;
      }
    }
    messageApi.success(
      editingApplication ? "客户端与安装包已更新" : "客户端与安装包已创建",
    );
    setApplicationPackageFiles([]);
    setEditingApplication(undefined);
    loadTab("content");
  };

  const openTicket = async (ticket: Ticket) => {
    setSelectedTicket(ticket);
    replyForm.setFieldsValue({
      body: "",
      status: ticket.status === "closed" ? "closed" : "pending",
    });
    const result = await HttpUtil.get<TicketMessage[]>(
      `/panel/api/commercial/tickets/${ticket.id}/messages`,
      undefined,
      { silent: true },
    );
    if (result.success && result.obj) setTicketMessages(result.obj);
  };
  const replyTicket = async () => {
    if (!selectedTicket) return;
    const values = await replyForm.validateFields();
    const result = await HttpUtil.post(
      `/panel/api/commercial/tickets/${selectedTicket.id}/reply`,
      values,
      jsonOptions(),
    );
    if (result.success) {
      messageApi.success("回复已发送");
      replyForm.resetFields(["body"]);
      await openTicket({ ...selectedTicket, status: values.status });
      loadTab("tickets");
    }
  };

  const openCoupon = (row?: Coupon) => {
    setEditingCoupon(row || null);
    couponForm.setFieldsValue(
      row
        ? {
            code: row.code,
            kind: row.kind,
            value: row.value,
            minimumYuan: row.minimumFen / 100,
            maxRedemptions: row.maxRedemptions,
            startsAt: row.startsAt || "",
            expiresAt: row.expiresAt || "",
            active: row.active,
          }
        : {
            code: "",
            kind: "fixed",
            value: 100,
            minimumYuan: 0,
            maxRedemptions: 0,
            startsAt: "",
            expiresAt: "",
            active: true,
          },
    );
  };
  const saveCoupon = async () => {
    const values = await couponForm.validateFields();
    const result = await HttpUtil.post(
      "/panel/api/commercial/coupons",
      {
        ...values,
        id: editingCoupon?.id || "",
        minimumFen: Math.round(values.minimumYuan * 100),
        startsAt: values.startsAt || null,
        expiresAt: values.expiresAt || null,
        redeemedCount: editingCoupon?.redeemedCount || 0,
      },
      jsonOptions(),
    );
    if (result.success) {
      messageApi.success(editingCoupon ? "优惠券已更新" : "优惠券已创建");
      setEditingCoupon(undefined);
      loadTab("marketing");
    }
  };

  const retryOrder = async (orderID: string) => {
    const result = await HttpUtil.post(
      `/panel/api/commercial/orders/${orderID}/retry`,
    );
    if (result.success) {
      messageApi.success("开通任务已重新排队");
      loadTab("orders");
    }
  };
  const savePaymentSettings = async (values: PaymentSettingsFormValues) => {
    const { password, twoFactorCode, ...payload } = values;
    const result = await HttpUtil.put(
      "/panel/api/commercial/payment-settings",
      payload,
      jsonOptions({
        "X-Admin-Password": password,
        "X-Admin-2FA": twoFactorCode || "",
      }),
    );
    if (result.success) {
      messageApi.success("支付设置已保存");
      await loadTab("settings");
      return true;
    }
    return false;
  };

  const createGiftCards = () => {
    let password = "";
    let twoFactorCode = "";
    let valueFen = 1000;
    let count = 1;
    Modal.confirm({
      title: "发行礼品卡",
      content: (
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Space.Compact block>
            <Button disabled>¥</Button>
            <InputNumber
              min={1}
              defaultValue={10}
              onChange={(value) => {
                valueFen = Math.round(Number(value || 0) * 100);
              }}
              style={{ width: "100%" }}
            />
          </Space.Compact>
          <Space.Compact block>
            <Button disabled>数量</Button>
            <InputNumber
              min={1}
              max={100}
              defaultValue={1}
              onChange={(value) => {
                count = Number(value || 1);
              }}
              style={{ width: "100%" }}
            />
          </Space.Compact>
          <Input.Password
            placeholder="管理员密码"
            onChange={(event) => {
              password = event.target.value;
            }}
          />
          <Input
            placeholder="2FA 验证码"
            onChange={(event) => {
              twoFactorCode = event.target.value;
            }}
          />
        </Space>
      ),
      okText: "发行",
      onOk: async () => {
        const result = await HttpUtil.post<{ codes: string[] }>(
          "/panel/api/commercial/gift-cards",
          { valueFen, count, expiresAt: "" },
          jsonOptions({
            "X-Admin-Password": password,
            "X-Admin-2FA": twoFactorCode,
          }),
        );
        if (result.success && result.obj) {
          Modal.info({
            title: "请立即复制兑换码",
            content: (
              <Paragraph copyable code>
                {result.obj.codes.join("\n")}
              </Paragraph>
            ),
            width: 620,
          });
          loadTab("marketing");
        }
      },
    });
  };
  const settleCommission = (row: Commission) => {
    let password = "";
    let twoFactorCode = "";
    Modal.confirm({
      title: "结算邀请佣金",
      content: (
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Text>将 {money(row.amountFen)} 计入邀请人余额。</Text>
          <Input.Password
            placeholder="管理员密码"
            onChange={(event) => {
              password = event.target.value;
            }}
          />
          <Input
            placeholder="2FA 验证码"
            onChange={(event) => {
              twoFactorCode = event.target.value;
            }}
          />
        </Space>
      ),
      okText: "确认结算",
      onOk: async () => {
        const result = await HttpUtil.post(
          `/panel/api/commercial/commissions/${row.id}/settle`,
          {},
          jsonOptions({
            "X-Admin-Password": password,
            "X-Admin-2FA": twoFactorCode,
          }),
        );
        if (result.success) {
          messageApi.success("佣金已结算");
          loadTab("marketing");
        }
      },
    });
  };

  const changeAdminRole = (row: AdminRole) => {
    let role = row.role;
    let password = "";
    let twoFactorCode = "";
    const roleOptions = [
      { value: "owner", label: "所有者" },
      { value: "administrator", label: "管理员" },
      { value: "finance", label: "财务" },
      { value: "support", label: "客服" },
      { value: "node_operator", label: "节点运维" },
      { value: "read_only_auditor", label: "只读审计" },
    ];
    Modal.confirm({
      title: `调整 ${row.username} 的角色`,
      content: (
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Select
            defaultValue={row.role}
            options={roleOptions}
            onChange={(value) => {
              role = value;
            }}
            style={{ width: "100%" }}
          />
          <Input.Password
            placeholder="管理员密码"
            onChange={(event) => {
              password = event.target.value;
            }}
          />
          <Input
            placeholder="2FA 验证码"
            onChange={(event) => {
              twoFactorCode = event.target.value;
            }}
          />
        </Space>
      ),
      okText: "保存角色",
      onOk: async () => {
        const result = await HttpUtil.put(
          `/panel/api/commercial/roles/${row.userId}`,
          { role },
          jsonOptions({
            "X-Admin-Password": password,
            "X-Admin-2FA": twoFactorCode,
          }),
        );
        if (result.success) {
          messageApi.success("管理员角色已更新");
          loadTab("roles");
        }
      },
    });
  };

  const customerColumns: TableProps<Customer>["columns"] = [
    {
      title: "客户",
      render: (_, row) => (
        <div>
          <Space>
            <strong>{row.displayName || row.email.split("@")[0]}</strong>
            {row.systemAdmin ? <Tag color="gold">默认管理员</Tag> : null}
          </Space>
          <br />
          <Text type="secondary">{row.email}</Text>
        </div>
      ),
    },
    {
      title: "订阅套餐",
      render: (_, row) =>
        row.subscription ? (
          <div>
            <strong>{row.subscription.plan.name}</strong>
            <br />
            <Text type="secondary">
              到期：{date(row.subscription.entitlement.expiresAt)}
            </Text>
          </div>
        ) : (
          <Tag>未开通</Tag>
        ),
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (value: string) => <Tag color={statusColor(value)}>{value}</Tag>,
    },
    { title: "余额", dataIndex: "balanceFen", render: money },
    { title: "注册时间", dataIndex: "createdAt", render: date },
    {
      title: "操作",
      render: (_, row) => (
        <Space wrap>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => openCustomer(row)}
          >
            资料
          </Button>
          <Button
            size="small"
            type="primary"
            ghost
            onClick={() => openSubscription(row)}
          >
            {row.subscription ? "管理订阅" : "手动开通"}
          </Button>
          <Button
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => deleteCustomers([row.id])}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];
  const orderColumns: TableProps<Order>["columns"] = [
    { title: "订单号", dataIndex: "outTradeNo" },
    {
      title: "类型",
      dataIndex: "orderKind",
      render: (value: Order["orderKind"]) =>
        ({ purchase: "新购", renewal: "续费", upgrade: "升级" })[
          value || "purchase"
        ],
    },
    { title: "客户 ID", dataIndex: "customerId", ellipsis: true },
    {
      title: "收款构成",
      render: (_, row) => (
        <Space orientation="vertical" size={0}>
          <Text>渠道 {money(row.paidFen)}</Text>
          {row.balancePaidFen > 0 && (
            <Text type="secondary">余额 {money(row.balancePaidFen)}</Text>
          )}
        </Space>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (value: string) => <Tag color={statusColor(value)}>{value}</Tag>,
    },
    {
      title: "异常",
      dataIndex: "failureReason",
      ellipsis: true,
      render: (value: string) => value || "—",
    },
    { title: "创建时间", dataIndex: "createdAt", render: date },
    {
      title: "操作",
      render: (_, row) => (
        <Space>
          {["paid", "provisioning", "provisioning_failed"].includes(
            row.status,
          ) && (
            <Popconfirm
              title="确认重新执行开通任务？"
              onConfirm={() => retryOrder(row.id)}
            >
              <Button size="small" icon={<ReloadOutlined />}>
                重新开通
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const pageClass = useMemo(
    () =>
      ["commercial-page", isDark ? "is-dark" : "", isUltra ? "is-ultra" : ""]
        .filter(Boolean)
        .join(" "),
    [isDark, isUltra],
  );
  const tabItems = [
    {
      key: "overview",
      label: "总览",
      icon: <BankOutlined />,
      children: <OverviewView data={overview} />,
    },
    {
      key: "customers",
      label: "客户",
      icon: <TeamOutlined />,
      children: (
        <>
          <div className="commercial-toolbar">
            <Button
              type="primary"
              icon={<UserAddOutlined />}
              onClick={openCreateCustomer}
            >
              手动创建客户
            </Button>
            <CustomerBulkDeleteButton
              selectedCount={selectedCustomerIDs.length}
              onEmpty={() => messageApi.info("请先勾选需要删除的客户")}
              onDelete={() => deleteCustomers(selectedCustomerIDs)}
            />
            <Input.Search
              allowClear
              placeholder="搜索邮箱、名称或邀请码"
              defaultValue={customerSearch}
              onSearch={(value) => {
                setCustomerPage(1);
                setCustomerSearch(value.trim());
              }}
              style={{ width: 270 }}
            />
            <Select
              value={customerStatus}
              onChange={(value) => {
                setCustomerPage(1);
                setCustomerStatus(value);
              }}
              style={{ width: 130 }}
              options={[
                { value: "", label: "全部状态" },
                { value: "active", label: "正常" },
                { value: "suspended", label: "暂停" },
                { value: "closed", label: "关闭" },
              ]}
            />
            <Text type="secondary">
              支持支付回调异常后的人工开通、改套餐和彻底清理白号
            </Text>
          </div>
          <Table
            rowKey="id"
            dataSource={customers}
            columns={customerColumns}
            pagination={{
              current: customerPage,
              pageSize: 100,
              total: customerTotal,
              showSizeChanger: false,
              showTotal: (total) => `共 ${total} 个客户`,
              onChange: setCustomerPage,
            }}
            rowSelection={{
              preserveSelectedRowKeys: true,
              selectedRowKeys: selectedCustomerIDs,
              onChange: (keys) => setSelectedCustomerIDs(keys.map(String)),
              getCheckboxProps: (row) => ({ name: row.email }),
            }}
          />
        </>
      ),
    },
    {
      key: "lines",
      label: "线路中心",
      icon: <ApartmentOutlined />,
      children: <LineCenterPane refreshToken={lineReloadToken} />,
    },
    {
      key: "plans",
      label: "套餐",
      icon: <SafetyCertificateOutlined />,
      children: (
        <PlansTable
          data={plans}
          onCreate={() => openPlan()}
          onEdit={openPlan}
        />
      ),
    },
    {
      key: "orders",
      label: "订单",
      icon: <ShoppingCartOutlined />,
      children: (
        <Table rowKey="id" dataSource={orders} columns={orderColumns} />
      ),
    },
    {
      key: "content",
      label: "公告与教程",
      icon: <NotificationOutlined />,
      children: (
        <ContentManager
          notices={notices}
          articles={articles}
          applications={applications}
          onEditNotice={(row) => openContent("notice", row)}
          onCreateNotice={() => openContent("notice")}
          onEditArticle={(row) => openContent("article", row)}
          onCreateArticle={() => openContent("article")}
          onEditApplication={openApplication}
          onCreateApplication={() => openApplication()}
        />
      ),
    },
    {
      key: "tickets",
      label: "工单",
      icon: <CustomerServiceOutlined />,
      children: <TicketTable data={tickets} onOpen={openTicket} />,
    },
    {
      key: "marketing",
      label: "营销与结算",
      icon: <GiftOutlined />,
      children: (
        <MarketingView
          coupons={coupons}
          giftCards={giftCards}
          commissions={commissions}
          onCreateCoupon={() => openCoupon()}
          onEditCoupon={openCoupon}
          onIssueGiftCards={createGiftCards}
          onSettle={settleCommission}
        />
      ),
    },
    {
      key: "settings",
      label: "支付设置",
      icon: <SettingOutlined />,
      children: (
        <PaymentSettingsView settings={settings} onSave={savePaymentSettings} />
      ),
    },
    {
      key: "roles",
      label: "权限角色",
      icon: <TeamOutlined />,
      children: <RolesTable data={adminRoles} onChange={changeAdminRole} />,
    },
    {
      key: "audit",
      label: "审计日志",
      icon: <AuditOutlined />,
      children: <AuditTable data={audit} />,
    },
  ];

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      {modalContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content id="content-layout" className="content-area">
            <Alert
              type="info"
              showIcon
              title="客户删除、人工维护订阅、敏感设置、礼品卡、佣金结算和角色变更需要重新验证管理员密码与 2FA。"
            />
            <Card className="commercial-workspace">
              <Spin spinning={loading}>
                <Tabs
                  animated={false}
                  activeKey={activeTab}
                  onChange={setActiveTab}
                  items={tabItems}
                  tabBarExtraContent={
                    <Button
                      icon={<ReloadOutlined />}
                      loading={loading}
                      aria-label="刷新"
                      title="刷新"
                      onClick={refreshActiveTab}
                    />
                  }
                />
              </Spin>
            </Card>
          </Layout.Content>
        </Layout>
      </Layout>

      <Modal
        open={Boolean(editingCustomer)}
        title="编辑客户"
        okText="保存"
        onOk={saveCustomer}
        onCancel={() => setEditingCustomer(null)}
        destroyOnHidden
      >
        <Form form={customerForm} layout="vertical">
          <Form.Item
            name="status"
            label="账户状态"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                { value: "active", label: "正常" },
                { value: "suspended", label: "暂停" },
                { value: "closed", label: "关闭" },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="balanceYuan"
            label="账户余额（元）"
            rules={[{ required: true }]}
          >
            <InputNumber min={0} precision={2} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={creatingCustomer}
        title="手动创建客户"
        okText="创建客户"
        onOk={saveCreatedCustomer}
        onCancel={() => setCreatingCustomer(false)}
        destroyOnHidden
      >
        <Alert
          type="info"
          showIcon
          title="管理员创建的客户无需邮箱验证码，创建后可直接登录前台。"
          style={{ marginBottom: 16 }}
        />
        <Form form={customerCreateForm} layout="vertical">
          <Form.Item
            name="email"
            label="邮箱地址"
            extra="允许的邮箱后缀以“安全设置”中的白名单为准。"
            rules={[{ required: true }, { type: "email" }]}
          >
            <Input placeholder="customer@example.com" autoComplete="off" />
          </Form.Item>
          <Form.Item name="displayName" label="显示名称">
            <Input maxLength={80} />
          </Form.Item>
          <Form.Item
            name="password"
            label="初始密码"
            extra="至少 10 位，并包含大写字母、小写字母和数字。"
            rules={[
              { required: true },
              { min: 10 },
              {
                pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/,
                message: "密码强度不足",
              },
            ]}
          >
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="locale" label="语言">
                <Select
                  options={locales.map((value) => ({ value, label: value }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="账户状态">
                <Select
                  options={[
                    { value: "active", label: "正常" },
                    { value: "suspended", label: "暂停" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={Boolean(subscriptionCustomer)}
        title={`${subscriptionCustomer?.subscription ? "管理" : "手动开通"}订阅 · ${subscriptionCustomer?.email || ""}`}
        width={720}
        onCancel={() => setSubscriptionCustomer(null)}
        destroyOnHidden
        footer={
          <div className="subscription-modal-footer">
            <div>
              {subscriptionCustomer?.subscription ? (
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => deleteSubscription(subscriptionCustomer)}
                >
                  删除订阅
                </Button>
              ) : null}
            </div>
            <Space>
              <Button onClick={() => setSubscriptionCustomer(null)}>
                取消
              </Button>
              <Button type="primary" onClick={saveSubscription}>
                {subscriptionCustomer?.subscription ? "保存订阅" : "立即开通"}
              </Button>
            </Space>
          </div>
        }
      >
        <Alert
          type="warning"
          showIcon
          title="此处直接同步面板客户端，可用于支付宝已到账但回调或自动开通失败的人工兜底。"
          style={{ marginBottom: 16 }}
        />
        <Form form={subscriptionForm} layout="vertical">
          <Form.Item name="planId" label="套餐" rules={[{ required: true }]}>
            <Select
              showSearch
              optionFilterProp="label"
              options={plans.map((item) => ({
                value: item.plan.id,
                label: `${item.plan.name}${item.plan.active ? "" : "（已下架）"}`,
              }))}
              onChange={selectSubscriptionPlan}
            />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="expiresAt" label="到期时间">
                <DatePicker showTime style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="trafficGB"
                label="流量额度（GB）"
                rules={[{ required: true }]}
              >
                <InputNumber min={0} precision={2} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="deviceLimit"
                label="设备/IP 上限"
                rules={[{ required: true }]}
              >
                <InputNumber min={0} max={1000} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item
                name="trafficMultiplier"
                label="流量计费倍率"
                extra="1 表示实际使用 1 GB 扣除 1 GB；2 表示扣除 2 GB。"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={0.1}
                  max={100}
                  step={0.1}
                  precision={2}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="uploadLimitMbps"
                label="上传限速（Mbps）"
                extra="0 表示不限速"
              >
                <InputNumber min={0} max={100000} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="downloadLimitMbps"
                label="下载限速（Mbps）"
                extra="0 表示不限速"
              >
                <InputNumber min={0} max={100000} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="resetTraffic"
            label="保存后立即清零已用流量"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Divider>管理员重新验证</Divider>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                name="password"
                label="管理员密码"
                rules={[{ required: true }]}
              >
                <Input.Password autoComplete="current-password" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="twoFactorCode" label="2FA 验证码（如已启用）">
                <Input maxLength={8} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={editingPlan !== undefined}
        title={editingPlan ? "编辑套餐" : "新建套餐"}
        okText="保存套餐"
        onOk={savePlan}
        onCancel={() => setEditingPlan(undefined)}
        width={860}
        destroyOnHidden
      >
        <Form form={planForm} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="套餐名称"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="slug"
                label="唯一标识"
                rules={[
                  { required: true },
                  {
                    pattern: /^[a-z0-9-]+$/,
                    message: "仅限小写字母、数字和连字符",
                  },
                ]}
              >
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="套餐说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Divider>前台套餐详情</Divider>
          <Alert
            type="info"
            showIcon
            title="这里的内容会直接显示在用户端套餐卡片中。"
            description="填写“包含”会显示勾选，填写“不包含”会显示短横线；也可以填写“精选线路”“1080P直播”等自定义说明。"
            style={{ marginBottom: 16 }}
          />
          <Row gutter={16}>
            {planDisplayBenefitFields.map((field) => (
              <Col span={8} key={field.key}>
                <Form.Item
                  name={["displayBenefits", field.key]}
                  label={field.label}
                  rules={[{ max: 120, message: "最多 120 个字符" }]}
                >
                  <Input placeholder="包含 / 不包含 / 自定义说明" />
                </Form.Item>
              </Col>
            ))}
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="trafficGB"
                label="流量（GB）"
                rules={[{ required: true }]}
              >
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="deviceLimit"
                label="设备/IP 上限"
                rules={[{ required: true }]}
              >
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="capacity" label="最多可售名额（0 不限）">
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="trafficMultiplier"
                label="流量计费倍率"
                extra="1x 为正常扣量，2x 节点使用 1 GB 将扣除 2 GB。"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={0.1}
                  max={100}
                  step={0.1}
                  precision={2}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="uploadLimitMbps"
                label="上传限速（Mbps）"
                extra="0 表示不限速"
              >
                <InputNumber min={0} max={100000} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="downloadLimitMbps"
                label="下载限速（Mbps）"
                extra="0 表示不限速"
              >
                <InputNumber min={0} max={100000} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="resetCycle" label="流量重置周期">
                <Select
                  options={[
                    "never",
                    "daily",
                    "weekly",
                    "monthly",
                    "quarterly",
                  ].map((value) => ({ value, label: value }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="nodeGroup"
                label="权限组标识"
                extra="用于识别套餐权限；实际可用范围由下方入站绑定决定。"
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="visibility" label="可见性">
                <Select
                  options={[
                    { value: "public", label: "公开售卖" },
                    { value: "hidden", label: "隐藏" },
                    { value: "invite", label: "仅邀请" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={24}>
              <Form.Item
                name="lineGroupIds"
                label="套餐线路组"
                extra="用户自动获得已选线路组内的全部健康节点。"
              >
                <Select
                  mode="multiple"
                  placeholder="选择线路中心中的分组"
                  options={lineGroups
                    .filter((group) => group.active)
                    .map((group) => ({ value: group.id, label: group.name }))}
                />
              </Form.Item>
            </Col>
            <Col span={16}>
              <Form.Item
                name="provisionInboundIds"
                label="高级：直接绑定入站 ID（逗号分隔）"
                extra="仅用于兼容手工管理的本机入站；普通套餐请使用上方线路组。"
              >
                <Input placeholder="例如：1,2,6" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="sortOrder" label="排序">
                <InputNumber style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Space size="large" wrap>
            <Form.Item name="active" label="上架" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item
              name="renewable"
              label="允许用户续费"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
            <Form.Item
              name="upgradable"
              label="允许升级到更高套餐"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Space>
          <Divider>住宅中转</Divider>
          <Alert
            type="info"
            showIcon
            title="启用后，用户可把套餐内指定线路的出口切换到自己提供的 SOCKS5 住宅代理。"
            description="每条中转只影响该用户选中的一条线路，使用流量仍计入套餐额度；当前仅支持主控服务器上的线路。"
            style={{ marginBottom: 16 }}
          />
          <Row gutter={16} align="middle">
            <Col span={8}>
              <Form.Item
                name="residentialRelayEnabled"
                label="允许住宅中转"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                noStyle
                shouldUpdate={(previous, current) =>
                  previous.residentialRelayEnabled !==
                  current.residentialRelayEnabled
                }
              >
                {({ getFieldValue }) => (
                  <Form.Item
                    name="residentialRelayLimit"
                    label="最多中转线路"
                    rules={
                      getFieldValue("residentialRelayEnabled")
                        ? [
                            {
                              required: true,
                              type: "number",
                              min: 1,
                              message: "请设置 1–20 条可中转线路",
                            },
                          ]
                        : undefined
                    }
                  >
                    <InputNumber
                      min={getFieldValue("residentialRelayEnabled") ? 1 : 0}
                      max={20}
                      disabled={!getFieldValue("residentialRelayEnabled")}
                      style={{ width: "100%" }}
                    />
                  </Form.Item>
                )}
              </Form.Item>
            </Col>
          </Row>
        </Form>
        <Divider>计费周期与价格</Divider>
        <Space orientation="vertical" style={{ width: "100%" }}>
          {planPrices.map((price, index) => (
            <Row gutter={12} key={price.id || index} align="middle">
              <Col span={7}>
                <Select
                  value={price.billingPeriod}
                  style={{ width: "100%" }}
                  options={[
                    "monthly",
                    "quarterly",
                    "half_yearly",
                    "yearly",
                    "multi_year",
                    "one_time",
                  ].map((value) => ({ value, label: value }))}
                  onChange={(value) =>
                    setPlanPrices((rows) =>
                      rows.map((row, rowIndex) =>
                        rowIndex === index
                          ? { ...row, billingPeriod: value }
                          : row,
                      ),
                    )
                  }
                />
              </Col>
              <Col span={5}>
                <Space.Compact block>
                  <InputNumber
                    value={price.months}
                    min={0}
                    style={{ width: "100%" }}
                    onChange={(value) =>
                      setPlanPrices((rows) =>
                        rows.map((row, rowIndex) =>
                          rowIndex === index
                            ? { ...row, months: Number(value || 0) }
                            : row,
                        ),
                      )
                    }
                  />
                  <Button disabled>个月</Button>
                </Space.Compact>
              </Col>
              <Col span={6}>
                <Space.Compact block>
                  <Button disabled>¥</Button>
                  <InputNumber
                    value={price.amountFen / 100}
                    min={0}
                    precision={2}
                    style={{ width: "100%" }}
                    onChange={(value) =>
                      setPlanPrices((rows) =>
                        rows.map((row, rowIndex) =>
                          rowIndex === index
                            ? {
                                ...row,
                                amountFen: Math.round(Number(value || 0) * 100),
                              }
                            : row,
                        ),
                      )
                    }
                  />
                </Space.Compact>
              </Col>
              <Col span={3}>
                <Switch
                  checked={price.active}
                  checkedChildren="启用"
                  unCheckedChildren="停用"
                  onChange={(value) =>
                    setPlanPrices((rows) =>
                      rows.map((row, rowIndex) =>
                        rowIndex === index ? { ...row, active: value } : row,
                      ),
                    )
                  }
                />
              </Col>
              <Col span={3}>
                <Button
                  danger
                  disabled={planPrices.length === 1}
                  onClick={() =>
                    setPlanPrices((rows) =>
                      rows.filter((_, rowIndex) => rowIndex !== index),
                    )
                  }
                >
                  移除
                </Button>
              </Col>
            </Row>
          ))}
        </Space>
        <Button
          type="dashed"
          block
          icon={<PlusOutlined />}
          style={{ marginTop: 12 }}
          onClick={() =>
            setPlanPrices((rows) => [
              ...rows,
              {
                id: "",
                planId: editingPlan?.plan.id || "",
                billingPeriod: "monthly",
                months: 1,
                amountFen: 0,
                active: true,
              },
            ])
          }
        >
          添加价格
        </Button>
      </Modal>

      <Modal
        open={Boolean(contentEditor)}
        title={`${contentEditor?.row ? "编辑" : "新建"}${contentEditor?.kind === "notice" ? "公告" : "教程"}`}
        okText="保存"
        onOk={saveContent}
        onCancel={() => setContentEditor(null)}
        width={900}
        destroyOnHidden
      >
        <Form form={contentForm} layout="vertical">
          <Row gutter={16}>
            <Col span={10}>
              <Form.Item
                name="slug"
                label="唯一标识"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={8}>
              {contentEditor?.kind === "notice" ? (
                <Form.Item name="level" label="公告级别">
                  <Select
                    options={["info", "success", "warning", "error"].map(
                      (value) => ({ value, label: value }),
                    )}
                  />
                </Form.Item>
              ) : (
                <Form.Item
                  name="category"
                  label="教程分类"
                  rules={[{ required: true }]}
                >
                  <Input placeholder="Windows / macOS / iOS" />
                </Form.Item>
              )}
            </Col>
            <Col span={6}>
              <Form.Item
                name="published"
                label="立即发布"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          {contentEditor?.kind === "article" && (
            <Form.Item name="sortOrder" label="排序">
              <InputNumber />
            </Form.Item>
          )}
          <Tabs
            items={locales.map((locale) => ({
              key: locale,
              label: locale,
              children: (
                <>
                  <Form.Item
                    name={["titles", locale]}
                    label={`${locale} 标题`}
                    rules={
                      locale === "zh-CN" || locale === "en-US"
                        ? [{ required: true }]
                        : undefined
                    }
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name={["contents", locale]}
                    label={`${locale} 正文`}
                    rules={
                      locale === "zh-CN" || locale === "en-US"
                        ? [{ required: true }]
                        : undefined
                    }
                  >
                    <Input.TextArea rows={7} />
                  </Form.Item>
                </>
              ),
            }))}
          />
        </Form>
      </Modal>

      <Modal
        open={editingApplication !== undefined}
        title={editingApplication ? "编辑客户端下载" : "新建客户端下载"}
        okText="保存"
        onOk={saveApplication}
        onCancel={() => {
          setEditingApplication(undefined);
          setApplicationPackageFiles([]);
        }}
        width={680}
        destroyOnHidden
      >
        <Form form={applicationForm} layout="vertical">
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="客户端名称"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="slug"
                label="唯一标识"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="platform"
            label="支持平台"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                "Windows",
                "macOS",
                "Android",
                "iOS",
                "Linux",
                "Windows / macOS / Linux",
              ].map((value) => ({ value, label: value }))}
              showSearch
            />
          </Form.Item>
          <Form.Item
            label="安装包"
            required={!editingApplication?.packageFileName}
            extra="支持 EXE、MSI、ZIP、7Z、DMG、PKG、APK、IPA、AppImage、DEB、RPM 和 TAR，单个文件最大 1 GB。保存后用户可从本站直接下载。"
          >
            <Upload
              accept=".exe,.msi,.zip,.7z,.dmg,.pkg,.apk,.ipa,.AppImage,.deb,.rpm,.tar.gz,.tar.xz,.tar.zst"
              maxCount={1}
              beforeUpload={() => false}
              fileList={applicationPackageFiles}
              onChange={({ fileList }) =>
                setApplicationPackageFiles(fileList.slice(-1))
              }
            >
              <Button icon={<UploadOutlined />}>
                {editingApplication?.packageFileName
                  ? "替换安装包"
                  : "选择安装包"}
              </Button>
            </Upload>
            {editingApplication?.packageFileName &&
              applicationPackageFiles.length === 0 && (
                <Text type="secondary">
                  当前文件：{editingApplication.packageFileName}（
                  {fileSize(editingApplication.packageSize)}）
                </Text>
              )}
          </Form.Item>
          <Form.Item name="description" label="用户端说明">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Space size="large">
            <Form.Item
              name="active"
              label="在用户端显示"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
            <Form.Item name="sortOrder" label="排序">
              <InputNumber />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      <Modal
        open={Boolean(selectedTicket)}
        title={selectedTicket?.subject}
        footer={null}
        onCancel={() => setSelectedTicket(null)}
        width={720}
        destroyOnHidden
      >
        {selectedTicket?.planName && (
          <Paragraph type="secondary">
            关联套餐：<Text strong>{selectedTicket.planName}</Text>
          </Paragraph>
        )}
        <div className="ticket-thread">
          {ticketMessages.map((item) => (
            <div
              key={item.id}
              className={`ticket-message is-${item.senderType}`}
            >
              <div>
                <Tag color={item.senderType === "admin" ? "blue" : "default"}>
                  {item.senderType === "admin" ? "客服" : "客户"}
                </Tag>
                <Text type="secondary">{date(item.createdAt)}</Text>
              </div>
              <Paragraph>{item.body}</Paragraph>
            </div>
          ))}
        </div>
        <Divider />
        <Form form={replyForm} layout="vertical">
          <Form.Item name="body" label="回复内容" rules={[{ required: true }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
          <Row gutter={12}>
            <Col flex="auto">
              <Form.Item name="status" label="回复后状态">
                <Select
                  options={[
                    { value: "pending", label: "等待客户回复" },
                    { value: "closed", label: "关闭工单" },
                    { value: "open", label: "保持开放" },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col>
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={replyTicket}
                style={{ marginTop: 30 }}
              >
                发送回复
              </Button>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={editingCoupon !== undefined}
        title={editingCoupon ? "编辑优惠券" : "新建优惠券"}
        okText="保存"
        onOk={saveCoupon}
        onCancel={() => setEditingCoupon(undefined)}
        destroyOnHidden
      >
        <Form form={couponForm} layout="vertical">
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                name="code"
                label="优惠码"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="kind" label="优惠类型">
                <Select
                  options={[
                    { value: "fixed", label: "固定金额（分）" },
                    { value: "percent", label: "折扣比例（万分比）" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="value"
            label="优惠值"
            rules={[{ required: true }]}
            extra="固定金额填分，例如 500 表示 ¥5；比例填万分比，例如 8500 表示 85 折。"
          >
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="minimumYuan" label="最低订单金额（元）">
                <InputNumber min={0} precision={2} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="maxRedemptions" label="最多使用次数（0 不限）">
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="startsAt" label="开始时间（RFC3339，可空）">
            <Input placeholder="2026-08-01T00:00:00Z" />
          </Form.Item>
          <Form.Item name="expiresAt" label="到期时间（RFC3339，可空）">
            <Input placeholder="2026-09-01T00:00:00Z" />
          </Form.Item>
          <Form.Item name="active" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </ConfigProvider>
  );
}

function OverviewView({ data }: { data: Overview | null }) {
  if (!data) return <div className="commercial-placeholder" />;
  const cards = [
    { title: "客户", value: data.customers, suffix: "人" },
    { title: "生效订阅", value: data.activeEntitlements, suffix: "个" },
    { title: "待支付订单", value: data.pendingOrders, suffix: "笔" },
    { title: "累计实收", value: money(data.revenueFen), suffix: "" },
    { title: "开放工单", value: data.openTickets, suffix: "个" },
    { title: "人工处理任务", value: data.manualJobs, suffix: "个" },
  ];
  return (
    <Row gutter={[16, 16]}>
      {cards.map((item) => (
        <Col xs={24} sm={12} lg={8} key={item.title}>
          <Card size="small" className="commercial-kpi">
            <Statistic
              title={item.title}
              value={item.value}
              suffix={item.suffix}
            />
          </Card>
        </Col>
      ))}
    </Row>
  );
}

function PlansTable({
  data,
  onCreate,
  onEdit,
}: {
  data: PlanItem[];
  onCreate: () => void;
  onEdit: (item: PlanItem) => void;
}) {
  const rows = data.map((item) => ({
    ...item.plan,
    prices: item.prices,
    item,
  }));
  return (
    <>
      <div className="commercial-toolbar">
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
          新建套餐
        </Button>
        <Text type="secondary">
          套餐内容、权限范围、计费倍率、带宽和价格都可在此维护
        </Text>
      </div>
      <Table
        rowKey="id"
        dataSource={rows}
        scroll={{ x: 1120 }}
        columns={[
          {
            title: "套餐",
            render: (_, row) => (
              <div>
                <strong>{row.name}</strong>
                <br />
                <Text type="secondary">{row.slug}</Text>
              </div>
            ),
          },
          {
            title: "流量",
            dataIndex: "trafficBytes",
            render: (value: number) => `${(value / GB).toFixed(0)} GB`,
          },
          {
            title: "倍率",
            dataIndex: "trafficMultiplierPermille",
            render: (value: number) =>
              `${((value || 1000) / 1000).toFixed(2).replace(/\.00$/, "")}x`,
          },
          {
            title: "带宽（上/下）",
            render: (_, row) =>
              `${row.uploadLimitMbps || "不限"}/${row.downloadLimitMbps || "不限"}${row.uploadLimitMbps || row.downloadLimitMbps ? " Mbps" : ""}`,
          },
          { title: "设备", dataIndex: "deviceLimit" },
          {
            title: "住宅中转",
            render: (_, row) =>
              row.residentialRelayEnabled ? (
                <Tag color="blue">最多 {row.residentialRelayLimit} 条</Tag>
              ) : (
                <Tag>未启用</Tag>
              ),
          },
          {
            title: "价格",
            dataIndex: "prices",
            render: (prices: PlanPrice[]) => (
              <Space wrap>
                {prices.map((price) => (
                  <Tag key={price.id} color={price.active ? "blue" : "default"}>
                    {price.billingPeriod} {money(price.amountFen)}
                  </Tag>
                ))}
              </Space>
            ),
          },
          { title: "可见性", dataIndex: "visibility" },
          {
            title: "状态",
            dataIndex: "active",
            render: (value: boolean) => (
              <Tag color={value ? "success" : "default"}>
                {value ? "已上架" : "已下架"}
              </Tag>
            ),
          },
          {
            title: "操作",
            render: (_, row) => (
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={() => onEdit(row.item)}
              >
                编辑
              </Button>
            ),
          },
        ]}
      />
    </>
  );
}

function ContentManager({
  notices,
  articles,
  applications,
  onCreateNotice,
  onEditNotice,
  onCreateArticle,
  onEditArticle,
  onCreateApplication,
  onEditApplication,
}: {
  notices: Notice[];
  articles: Article[];
  applications: Application[];
  onCreateNotice: () => void;
  onEditNotice: (row: Notice) => void;
  onCreateArticle: () => void;
  onEditArticle: (row: Article) => void;
  onCreateApplication: () => void;
  onEditApplication: (row: Application) => void;
}) {
  return (
    <Tabs
      items={[
        {
          key: "notices",
          label: `公告 (${notices.length})`,
          children: (
            <>
              <div className="commercial-toolbar">
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={onCreateNotice}
                >
                  新建公告
                </Button>
              </div>
              <Table
                rowKey="id"
                dataSource={notices}
                columns={[
                  { title: "标识", dataIndex: "slug" },
                  { title: "级别", dataIndex: "level" },
                  {
                    title: "标题",
                    dataIndex: "titleI18n",
                    render: (value: string) =>
                      parseI18n(value)["zh-CN"] ||
                      parseI18n(value)["en-US"] ||
                      "—",
                  },
                  {
                    title: "发布状态",
                    dataIndex: "published",
                    render: (value: boolean) => (
                      <Tag color={value ? "success" : "default"}>
                        {value ? "已发布" : "草稿"}
                      </Tag>
                    ),
                  },
                  { title: "更新", dataIndex: "updatedAt", render: date },
                  {
                    title: "操作",
                    render: (_, row) => (
                      <Button
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => onEditNotice(row)}
                      >
                        编辑
                      </Button>
                    ),
                  },
                ]}
              />
            </>
          ),
        },
        {
          key: "articles",
          label: `教程 (${articles.length})`,
          children: (
            <>
              <div className="commercial-toolbar">
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={onCreateArticle}
                >
                  新建教程
                </Button>
              </div>
              <Table
                rowKey="id"
                dataSource={articles}
                columns={[
                  { title: "标识", dataIndex: "slug" },
                  { title: "分类", dataIndex: "category" },
                  {
                    title: "标题",
                    dataIndex: "titleI18n",
                    render: (value: string) =>
                      parseI18n(value)["zh-CN"] ||
                      parseI18n(value)["en-US"] ||
                      "—",
                  },
                  {
                    title: "发布状态",
                    dataIndex: "published",
                    render: (value: boolean) => (
                      <Tag color={value ? "success" : "default"}>
                        {value ? "已发布" : "草稿"}
                      </Tag>
                    ),
                  },
                  {
                    title: "操作",
                    render: (_, row) => (
                      <Button
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => onEditArticle(row)}
                      >
                        编辑
                      </Button>
                    ),
                  },
                ]}
              />
            </>
          ),
        },
        {
          key: "applications",
          label: `客户端 (${applications.length})`,
          children: (
            <>
              <div className="commercial-toolbar">
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={onCreateApplication}
                >
                  新建客户端下载
                </Button>
                <Text type="secondary">
                  上传安装包后，用户可在“使用文档”中从本站直接下载。
                </Text>
              </div>
              <Table
                rowKey="id"
                dataSource={applications}
                columns={[
                  {
                    title: "客户端",
                    render: (_, row) => (
                      <div>
                        <strong>{row.name}</strong>
                        <br />
                        <Text type="secondary">{row.slug}</Text>
                      </div>
                    ),
                  },
                  { title: "平台", dataIndex: "platform" },
                  {
                    title: "安装包",
                    render: (_, row) =>
                      row.packageFileName ? (
                        <div>
                          <Text>{row.packageFileName}</Text>
                          <br />
                          <Text type="secondary">
                            {fileSize(row.packageSize)}
                          </Text>
                        </div>
                      ) : (
                        <Tag color="warning">未上传</Tag>
                      ),
                  },
                  {
                    title: "用户端状态",
                    render: (_, row) => (
                      <Tag
                        color={
                          row.active && row.packageFileName
                            ? "success"
                            : "default"
                        }
                      >
                        {row.active && row.packageFileName
                          ? "可下载"
                          : row.active
                            ? "等待上传"
                            : "已停用"}
                      </Tag>
                    ),
                  },
                  {
                    title: "操作",
                    render: (_, row) => (
                      <Button
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => onEditApplication(row)}
                      >
                        编辑
                      </Button>
                    ),
                  },
                ]}
              />
            </>
          ),
        },
      ]}
    />
  );
}

function TicketTable({
  data,
  onOpen,
}: {
  data: Ticket[];
  onOpen: (ticket: Ticket) => void;
}) {
  return (
    <Table
      rowKey="id"
      dataSource={data}
      columns={[
        { title: "主题", dataIndex: "subject" },
        { title: "客户", dataIndex: "customerId", ellipsis: true },
        {
          title: "关联套餐",
          dataIndex: "planName",
          render: (value?: string) => value || "—",
        },
        { title: "优先级", dataIndex: "priority" },
        {
          title: "状态",
          dataIndex: "status",
          render: (value: string) => (
            <Tag color={statusColor(value)}>{value}</Tag>
          ),
        },
        { title: "更新", dataIndex: "updatedAt", render: date },
        {
          title: "操作",
          render: (_, row) => (
            <Button size="small" onClick={() => onOpen(row)}>
              查看与回复
            </Button>
          ),
        },
      ]}
    />
  );
}

export function MarketingView({
  coupons,
  giftCards,
  commissions,
  onCreateCoupon,
  onEditCoupon,
  onIssueGiftCards,
  onSettle,
}: {
  coupons: Coupon[];
  giftCards: GiftCard[];
  commissions: Commission[];
  onCreateCoupon: () => void;
  onEditCoupon: (row: Coupon) => void;
  onIssueGiftCards: () => void;
  onSettle: (row: Commission) => void;
}) {
  const invitation = useInvitationSettings();
  const [invitationMessageApi, invitationMessageContextHolder] =
    message.useMessage();
  const saveInvitationSettings = async () => {
    try {
      await invitation.saveInvitationSettings();
      invitationMessageApi.success("邀请与佣金设置已保存");
    } catch (error) {
      invitationMessageApi.error(
        error instanceof Error ? error.message : "邀请与佣金设置保存失败",
      );
    }
  };
  return (
    <>
      {invitationMessageContextHolder}
      <Tabs
        items={[
          {
            key: "invitation-settings",
            label: "邀请&佣金设置",
            children: (
              <InvitationSettingsPane
                settings={invitation.invitationSettings}
                error={invitation.error}
                spinning={invitation.spinning}
                saveDisabled={invitation.saveDisabled}
                onChange={invitation.updateInvitationSettings}
                onSave={saveInvitationSettings}
              />
            ),
          },
          {
            key: "coupons",
            label: `优惠券 (${coupons.length})`,
            children: (
              <>
                <div className="commercial-toolbar">
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={onCreateCoupon}
                  >
                    新建优惠券
                  </Button>
                </div>
                <Table
                  rowKey="id"
                  dataSource={coupons}
                  columns={[
                    { title: "优惠码", dataIndex: "code" },
                    { title: "类型", dataIndex: "kind" },
                    { title: "优惠值", dataIndex: "value" },
                    {
                      title: "已使用 / 上限",
                      render: (_, row) =>
                        `${row.redeemedCount} / ${row.maxRedemptions || "不限"}`,
                    },
                    {
                      title: "状态",
                      dataIndex: "active",
                      render: (value: boolean) => (
                        <Tag color={value ? "success" : "default"}>
                          {value ? "启用" : "停用"}
                        </Tag>
                      ),
                    },
                    {
                      title: "操作",
                      render: (_, row) => (
                        <Button
                          size="small"
                          icon={<EditOutlined />}
                          onClick={() => onEditCoupon(row)}
                        >
                          编辑
                        </Button>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          {
            key: "gifts",
            label: `礼品卡 (${giftCards.length})`,
            children: (
              <>
                <div className="commercial-toolbar">
                  <Button
                    type="primary"
                    icon={<GiftOutlined />}
                    onClick={onIssueGiftCards}
                  >
                    发行礼品卡
                  </Button>
                </div>
                <Table
                  rowKey="id"
                  dataSource={giftCards}
                  columns={[
                    { title: "兑换码", dataIndex: "displayCode" },
                    { title: "面值", dataIndex: "valueFen", render: money },
                    { title: "状态", dataIndex: "status" },
                    { title: "发行时间", dataIndex: "createdAt", render: date },
                  ]}
                />
              </>
            ),
          },
          {
            key: "commissions",
            label: `邀请佣金 (${commissions.length})`,
            children: (
              <Table
                rowKey="id"
                dataSource={commissions}
                columns={[
                  { title: "邀请人", dataIndex: "inviterId", ellipsis: true },
                  { title: "被邀请人", dataIndex: "inviteeId", ellipsis: true },
                  { title: "金额", dataIndex: "amountFen", render: money },
                  {
                    title: "状态",
                    dataIndex: "status",
                    render: (value: string) => (
                      <Tag color={statusColor(value)}>{value}</Tag>
                    ),
                  },
                  {
                    title: "操作",
                    render: (_, row) =>
                      ["pending", "confirmed"].includes(row.status) && (
                        <Button
                          size="small"
                          type="primary"
                          onClick={() => onSettle(row)}
                        >
                          结算
                        </Button>
                      ),
                  },
                ]}
              />
            ),
          },
        ]}
      />
    </>
  );
}

function RolesTable({
  data,
  onChange,
}: {
  data: AdminRole[];
  onChange: (row: AdminRole) => void;
}) {
  const labels: Record<string, string> = {
    owner: "所有者",
    administrator: "管理员",
    finance: "财务",
    support: "客服",
    node_operator: "节点运维",
    read_only_auditor: "只读审计",
  };
  return (
    <>
      <Alert
        type="warning"
        showIcon
        title="只有所有者可以调整角色；变更时必须重新验证密码与 2FA。"
      />
      <Table
        rowKey="userId"
        dataSource={data}
        columns={[
          { title: "管理员 ID", dataIndex: "userId" },
          { title: "账号", dataIndex: "username" },
          {
            title: "当前角色",
            dataIndex: "role",
            render: (value: string) => (
              <Tag color={value === "owner" ? "gold" : "blue"}>
                {labels[value] || value}
              </Tag>
            ),
          },
          {
            title: "操作",
            render: (_, row) => (
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={() => onChange(row)}
              >
                调整角色
              </Button>
            ),
          },
        ]}
      />
    </>
  );
}

function AuditTable({ data }: { data: AuditLog[] }) {
  return (
    <Table
      rowKey="id"
      dataSource={data}
      columns={[
        { title: "时间", dataIndex: "createdAt", render: date },
        {
          title: "操作人",
          render: (_, row) => `${row.actorRole} #${row.actorUserId}`,
        },
        { title: "操作", dataIndex: "action" },
        {
          title: "对象",
          render: (_, row) => `${row.targetType}:${row.targetId}`,
        },
      ]}
    />
  );
}
