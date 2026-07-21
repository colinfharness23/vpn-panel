export interface Envelope<T> {
  success: boolean;
  msg: string;
  obj: T;
}

export interface Plan {
  id: string;
  slug: string;
  name: string;
  description: string;
  trafficBytes: number;
  deviceLimit: number;
  uploadLimitMbps?: number;
  downloadLimitMbps?: number;
  residentialRelayEnabled?: boolean;
  residentialRelayLimit?: number;
  resetCycle: string;
  nodeGroup: string;
  visibility: string;
  renewable: boolean;
  upgradable: boolean;
  active: boolean;
  sortOrder?: number;
  displayBenefits?: PlanDisplayBenefits;
}

export interface PlanDisplayBenefits {
  globalCoverage?: string;
  standardNodes?: string;
  advancedNodes?: string;
  premiumRoutes?: string;
  residentialIpSale?: string;
  socialMedia?: string;
  crossBorderWork?: string;
  liveStreaming?: string;
  uploadOptimization?: string;
  peakPriority?: string;
  failover?: string;
  support?: string;
}

export interface PlanPrice {
  id: string;
  planId?: string;
  billingPeriod: string;
  months: number;
  amountFen: number;
  active: boolean;
}

export interface PlanCatalogItem {
  plan: Plan;
  prices: PlanPrice[];
}

export interface LocalizedContent {
  id: string;
  slug: string;
  category?: string;
  level?: string;
  title: string;
  content: string;
}

export interface ClientApplication {
  id: string;
  slug: string;
  name: string;
  platform: string;
  description: string;
  packageFileName?: string;
  packageSize?: number;
  packageSha256?: string;
  downloadUrl?: string;
}

export interface PaymentMethod {
  code: string;
  name: string;
}

export interface GuestBootstrap {
  site: Record<string, string>;
  plans: PlanCatalogItem[];
  paymentMethods: PaymentMethod[];
  notices: LocalizedContent[];
  articles: LocalizedContent[];
  applications: ClientApplication[];
}

export interface GuestAuthConfig {
  site: Record<string, string>;
}

export interface Customer {
  id: string;
  email: string;
  displayName: string;
  locale: string;
  status: string;
  balanceFen: number;
  inviteCode: string;
  termsAcceptedAt?: string;
  termsVersion?: string;
}

export interface Entitlement {
  id: string;
  planId: string;
  status: string;
  trafficQuota: number;
  trafficUsed: number;
  deviceLimit: number;
  residentialRelayEnabled?: boolean;
  residentialRelayLimit?: number;
  startsAt: string;
  expiresAt?: string;
}

export interface SubscriptionOverview {
  entitlement: Entitlement;
  plan: Plan;
  usedBytes: number;
  links: {
    raw: string;
    clash: string;
    json: string;
  };
}

export interface Order {
  id: string;
  outTradeNo: string;
  orderKind?: "purchase" | "renewal" | "upgrade";
  entitlementId?: string;
  resultExpiresAt?: string;
  status: string;
  originalFen: number;
  discountFen: number;
  balancePaidFen: number;
  payableFen: number;
  paidFen: number;
  expiresAt: string;
  createdAt: string;
}

export interface Dashboard {
  customer: Customer;
  subscription?: SubscriptionOverview;
  invitation: {
    enabled: boolean;
    inviteCode: string;
    directInviteCount: number;
    commissionPercent: number;
    pendingFen: number;
    confirmedFen: number;
    settledFen: number;
    commissionFirstPaymentOnly: boolean;
    inviteCodesNeverExpire: boolean;
  };
  notices: LocalizedContent[];
  orders: Order[];
}

export interface PaymentIntent {
  provider: string;
  outTradeNo: string;
  qrCode: string;
  amountFen: number;
  expiresAt: string;
}

export interface PaymentPayload {
  intent: PaymentIntent;
  qrImage: string;
}

export interface Ticket {
  id: string;
  entitlementId?: string;
  planId?: string;
  planName?: string;
  subject: string;
  status: string;
  priority: string;
  createdAt: string;
}

export interface TicketMessage {
  id: string;
  ticketId: string;
  senderType: string;
  body: string;
  createdAt: string;
}

export interface CustomerSession {
  id: string;
  lastSeenAt: string;
  expiresAt: string;
  revokedAt?: string;
  createdAt: string;
}

export interface SessionList {
  currentSessionId: string;
  sessions: CustomerSession[];
}

export interface ResidentialRelayLine {
  id: number;
  name: string;
  protocol: string;
}

export interface ResidentialRelay {
  id: string;
  inboundId: number;
  lineName: string;
  protocol: string;
  name: string;
  host: string;
  port: number;
  username?: string;
  hasPassword: boolean;
  status: "active" | "pending";
  createdAt: string;
  links: string[];
}

export interface ResidentialRelayOverview {
  enabled: boolean;
  limit: number;
  lines: ResidentialRelayLine[];
  relays: ResidentialRelay[];
}
