export type PortalLocale = 'ar-EG' | 'en-US' | 'fa-IR' | 'zh-CN' | 'zh-TW' | 'ja-JP' | 'ru-RU' | 'vi-VN' | 'es-ES' | 'id-ID' | 'uk-UA' | 'tr-TR' | 'pt-BR';

export interface PortalCopy {
  home: string;
  subscription: string;
  plans: string;
  guides: string;
  clients: string;
  tickets: string;
  signIn: string;
  signOut: string;
  currentPlan: string;
  used: string;
  expires: string;
  daysLeft: string;
  manage: string;
  importSubscription: string;
  importDescription: string;
  copyLink: string;
  showQR: string;
  quickStart: string;
  chooseClient: string;
  importProfile: string;
  connect: string;
  officialDownload: string;
  buyNow: string;
  perMonth: string;
  noSubscription: string;
  login: string;
  register: string;
  reset: string;
  email: string;
  emailOrAdmin: string;
  password: string;
  code: string;
  sendCode: string;
  submit: string;
  createTicket: string;
  relatedPlan: string;
  relatedPlanHint: string;
  subject: string;
  body: string;
  orders: string;
  announcementDetails: string;
  heroBadge: string;
  heroTitle: string;
  heroDescription: string;
  browsePlans: string;
  readGuides: string;
  serviceTitle: string;
  serviceDescription: string;
  platformGuides: string;
  subscriptionFormats: string;
  selfService: string;
  benefitsTitle: string;
  benefitsDescription: string;
  fastTitle: string;
  fastDescription: string;
  privacyTitle: string;
  privacyDescription: string;
  simpleTitle: string;
  simpleDescription: string;
  greeting: string;
  verified: string;
  status: string;
  active: string;
  totalTraffic: string;
  trafficReset: string;
  importToClient: string;
  subscriptionLinkLabel: string;
  securityNote: string;
  clientStepDescription: string;
  importStepDescription: string;
  connectStepDescription: string;
  allGuides: string;
  billingNotice: string;
  recommended: string;
  traffic: string;
  devices: string;
  automaticActivation: string;
  guidesDescription: string;
  clientsDescription: string;
  signInForTickets: string;
  emptyTickets: string;
  deviceLimit: string;
  gmailOnly: string;
  inviteOptional: string;
  acceptTermsPrefix: string;
  termsOfService: string;
  termsRequired: string;
  readTerms: string;
  acceptTermsAction: string;
  alipayTitle: string;
  paymentTitle: string;
  choosePaymentMethod: string;
  choosePaymentMethodDescription: string;
  noPaymentMethods: string;
  orderLabel: string;
  paymentValidUntil: string;
  confirmDemoPayment: string;
  subscriptionQRTitle: string;
  subscriptionQRWarning: string;
  orderNumber: string;
  amount: string;
  createdAt: string;
  emptyOrders: string;
  registrationSuccess: string;
  loginSuccess: string;
  codeSent: string;
  demoPaymentSuccess: string;
  linkCopied: string;
  ticketSubmitted: string;
  quota: string;
  newPassword: string;
  confirmPassword: string;
  passwordRule: string;
  resetDescription: string;
  resetSuccess: string;
  passwordMismatch: string;
  billingPeriod: string;
  couponCode: string;
  rotateLink: string;
  rotateConfirm: string;
  linkRotated: string;
  continuePayment: string;
  cancelOrder: string;
  orderCancelled: string;
  ticketConversation: string;
  reply: string;
  accountSecurity: string;
  sessions: string;
  currentSession: string;
  revokeSession: string;
  balance: string;
  redeemGiftCard: string;
  giftCardCode: string;
  redeemedSuccess: string;
  renew: string;
  upgrade: string;
  renewalDescription: string;
  upgradeDescription: string;
  accountCenter: string;
  accountOverview: string;
  invitationRewards: string;
  inviteFriends: string;
  invitationDescription: string;
  invitationCode: string;
  invitationLink: string;
  copyInvitation: string;
  invitedUsers: string;
  commissionRate: string;
  pendingCommission: string;
  earnedCommission: string;
  commissionBalanceHint: string;
  firstPaymentRule: string;
  inviteCodePermanent: string;
  startHere: string;
  emptySubscriptionDescription: string;
  subscriptionReadyTitle: string;
  subscriptionReadyDescription: string;
  planHelpTitle: string;
  planHelpDescription: string;
  guideStartTitle: string;
  guideStartDescription: string;
  troubleshooting: string;
  troubleshootingDescription: string;
  clientPickerTitle: string;
  clientPickerDescription: string;
  afterInstallTitle: string;
  afterInstallDescription: string;
  supportCenterTitle: string;
  supportCenterDescription: string;
  telegramSupport: string;
  responseTimeTitle: string;
  responseTimeDescription: string;
  faqTitle: string;
  faqDescription: string;
  paymentSecureTitle: string;
  paymentSecureDescription: string;
  activationTitle: string;
  activationDescription: string;
  invitationCopied: string;
  changePassword: string;
  currentPassword: string;
  passwordChangeHint: string;
  passwordChanged: string;
}

const en: PortalCopy = {
  home: 'Home', subscription: 'My subscription', plans: 'Plans', guides: 'Setup guides', clients: 'Apps', tickets: 'Support', signIn: 'Sign in', signOut: 'Sign out', currentPlan: 'Current plan', used: 'Used', expires: 'Expires', daysLeft: 'days left', manage: 'Manage subscription', importSubscription: 'Import subscription', importDescription: 'Copy your private subscription link or scan the QR code in a supported app.', copyLink: 'Copy subscription link', showQR: 'Show QR code', quickStart: 'Quick start', chooseClient: 'Get an official app', importProfile: 'Import your subscription', connect: 'Choose a node and connect', officialDownload: 'Official download', buyNow: 'Choose plan', perMonth: '/ month', noSubscription: 'No active subscription yet', login: 'Sign in', register: 'Create account', reset: 'Reset password', email: 'Gmail address', emailOrAdmin: 'Gmail address or administrator username', password: 'Password', code: 'Verification code', sendCode: 'Send code', submit: 'Continue', createTicket: 'Create ticket', relatedPlan: 'Related plan', relatedPlanHint: 'Select the plan affected by this issue so support can identify it quickly.', subject: 'Subject', body: 'How can we help?', orders: 'Orders', announcementDetails: 'View details', heroBadge: 'Secure and reliable access', heroTitle: 'One subscription for a simpler, more reliable connection', heroDescription: 'Choose a plan, complete payment, and import your private subscription into a supported app in minutes.', browsePlans: 'Browse plans', readGuides: 'Read setup guides', serviceTitle: 'Everything you need to get connected', serviceDescription: 'Clear setup steps, official app links, and self-service subscription management in one place.', platformGuides: 'platform guides', subscriptionFormats: 'subscription formats', selfService: 'self-service access', benefitsTitle: 'Built for a smooth daily experience', benefitsDescription: 'From purchase to connection, every step stays clear and manageable.', fastTitle: 'Fast node access', fastDescription: 'Use the node group included with your plan and refresh your subscription whenever nodes change.', privacyTitle: 'Private by default', privacyDescription: 'Your subscription token can be rotated or revoked from your account if it is ever exposed.', simpleTitle: 'Simple on every device', simpleDescription: 'Follow platform-specific guides and open only official app stores or GitHub releases.', greeting: 'Hello', verified: 'Verified', status: 'Status', active: 'Active', totalTraffic: 'Total traffic', trafficReset: 'Traffic resets when the current period ends', importToClient: 'Import subscription into your VPN app', subscriptionLinkLabel: 'Private subscription link — do not share', securityNote: 'For account security, keep this link private and never publish or share it.', clientStepDescription: 'Download and install a supported official app', importStepDescription: 'Copy the subscription link and import it into the app', connectStepDescription: 'Choose a node and connect', allGuides: 'View all guides', billingNotice: 'All prices are charged in CNY. Access is provisioned automatically after payment.', recommended: 'Recommended', traffic: 'traffic', devices: 'devices', automaticActivation: 'Automatic activation after payment', guidesDescription: 'Choose your device and finish importing the subscription in a few minutes.', clientsDescription: 'Links open only official stores or GitHub releases; apps are never repackaged.', signInForTickets: 'Sign in to create and view support tickets', emptyTickets: 'No support tickets yet', deviceLimit: 'Device limit', gmailOnly: 'Registration is available only with a Gmail verification code', inviteOptional: 'Invitation code (optional)', acceptTermsPrefix: 'I have read and agree to', termsOfService: 'Terms of Service', termsRequired: 'Please read and agree to the Terms of Service before creating an account.', readTerms: 'Read terms', acceptTermsAction: 'Agree and continue', alipayTitle: 'Pay with Alipay', paymentTitle: 'Online payment', choosePaymentMethod: 'Choose a payment method', choosePaymentMethodDescription: 'Select one of the payment methods enabled by the site administrator.', noPaymentMethods: 'No payment methods are currently available', orderLabel: 'Order', paymentValidUntil: 'QR code valid until', confirmDemoPayment: 'Confirm demo payment', subscriptionQRTitle: 'Subscription QR code', subscriptionQRWarning: 'Do not share this QR code. Rotate your subscription link immediately if it is exposed.', orderNumber: 'Order number', amount: 'Amount', createdAt: 'Created', emptyOrders: 'No orders yet', registrationSuccess: 'Account created', loginSuccess: 'Signed in', codeSent: 'Verification code sent', demoPaymentSuccess: 'Demo payment confirmed. Provisioning has started.', linkCopied: 'Subscription link copied', ticketSubmitted: 'Ticket submitted', quota: 'Quota', newPassword: 'New password', confirmPassword: 'Confirm new password', passwordRule: 'Use at least 10 characters with uppercase, lowercase, and a number.', resetDescription: 'Verify your Gmail address, then set a new password. Your old password is not required.', resetSuccess: 'Password reset. Sign in with your new password.', passwordMismatch: 'The two passwords do not match', billingPeriod: 'Billing period', couponCode: 'Redeem code (optional)', rotateLink: 'Rotate private link', rotateConfirm: 'The old subscription link will stop working immediately. Continue?', linkRotated: 'A new subscription link has been created', continuePayment: 'Continue payment', cancelOrder: 'Cancel order', orderCancelled: 'Order cancelled', ticketConversation: 'Conversation', reply: 'Reply', accountSecurity: 'Account & security', sessions: 'Login sessions', currentSession: 'Current session', revokeSession: 'Sign out session', balance: 'Account balance', redeemGiftCard: 'Redeem gift card', giftCardCode: 'Gift card code', redeemedSuccess: 'Gift card redeemed', renew: 'Renew', upgrade: 'Upgrade plan', renewalDescription: 'Extend the current subscription from its existing expiry date.', upgradeDescription: 'Switch to the higher plan now and keep the remaining subscription time.', accountCenter: 'Personal center', accountOverview: 'Overview', invitationRewards: 'Invitation rewards', inviteFriends: 'Invite friends', invitationDescription: 'Share your personal invitation link. Rewards follow the current site policy and are credited to your account balance after settlement.', invitationCode: 'Invitation code', invitationLink: 'Personal invitation link', copyInvitation: 'Copy invitation link', invitedUsers: 'Friends invited', commissionRate: 'Reward rate', pendingCommission: 'Pending rewards', earnedCommission: 'Rewards earned', commissionBalanceHint: 'Account balance cannot be withdrawn and can only be used for plan purchases, renewals, and upgrades.', firstPaymentRule: 'Rewards are calculated on the invited user’s first successful payment.', inviteCodePermanent: 'Your invitation code does not expire.', startHere: 'Start here', emptySubscriptionDescription: 'Choose a plan first. After payment, your subscription link and setup steps will appear here automatically.', subscriptionReadyTitle: 'From purchase to connection', subscriptionReadyDescription: 'Choose a plan, complete payment, install an official app, then import your private subscription.', planHelpTitle: 'Choose with confidence', planHelpDescription: 'Compare traffic, device limit and billing period. Access is provisioned automatically after payment.', guideStartTitle: 'Three steps to connect', guideStartDescription: 'Install an official app, import the subscription, then choose a node and connect.', troubleshooting: 'Troubleshooting', troubleshootingDescription: 'Refresh the subscription first. If the issue continues, submit a ticket or contact Telegram support.', clientPickerTitle: 'Choose the app for your device', clientPickerDescription: 'Use desktop apps on Windows, macOS or Linux and an official mobile app on Android or iOS.', afterInstallTitle: 'After installation', afterInstallDescription: 'Open My subscription, copy the private link, and follow the setup guide for your platform.', supportCenterTitle: 'Support center', supportCenterDescription: 'Use tickets for account or subscription issues. Telegram is available for quick service notices and general help.', telegramSupport: 'Telegram support', responseTimeTitle: 'Ticket history stays in your account', responseTimeDescription: 'Replies remain attached to the ticket so you can continue the conversation later.', faqTitle: 'Before submitting a ticket', faqDescription: 'Include your device, app name and the exact error. Never include your password or full subscription link.', paymentSecureTitle: 'Secure checkout', paymentSecureDescription: 'Choose an available payment method after the payable amount is confirmed.', activationTitle: 'Automatic activation', activationDescription: 'Your subscription is provisioned after payment is confirmed.', invitationCopied: 'Invitation link copied', changePassword: 'Change password', currentPassword: 'Current password', passwordChangeHint: 'Confirm your current password, then choose a new password. Other sessions will be signed out.', passwordChanged: 'Password changed. Other sessions have been signed out.',
};

export const portalCopies: Record<PortalLocale, PortalCopy> = {
  'en-US': en,
  'zh-CN': { ...en, home: '首页', subscription: '我的订阅', plans: '套餐', guides: '使用教程', clients: '客户端', tickets: '工单', signIn: '登录', signOut: '退出登录', currentPlan: '当前套餐', used: '已使用', expires: '到期时间', daysLeft: '天后到期', manage: '管理订阅', importSubscription: '导入订阅', importDescription: '复制专属订阅链接，或在支持的客户端中扫描二维码。', copyLink: '复制订阅链接', showQR: '显示二维码', quickStart: '快速开始', chooseClient: '选择客户端', importProfile: '导入订阅链接', connect: '选择节点并连接', officialDownload: '下载', buyNow: '立即选购', perMonth: '/ 月', noSubscription: '当前没有生效中的订阅', login: '登录', register: '注册账号', reset: '找回密码', email: 'Gmail 邮箱', password: '登录密码', code: '邮箱验证码', sendCode: '发送验证码', submit: '继续', createTicket: '提交工单', subject: '工单主题', body: '请描述你遇到的问题', orders: '订单记录', announcementDetails: '查看详情', heroBadge: '安全稳定的连接服务', heroTitle: '一条订阅，轻松连接你的所有设备', heroDescription: '选择套餐、完成付款，再将专属订阅导入支持的客户端，几分钟即可开始使用。', browsePlans: '查看套餐', readGuides: '阅读教程', serviceTitle: '连接所需的信息，都在这里', serviceDescription: '平台教程、客户端下载与订阅管理集中呈现，每一步都清晰可查。', platformGuides: '个平台教程', subscriptionFormats: '种订阅格式', selfService: '自助服务', benefitsTitle: '从购买到连接，全程简单清晰', benefitsDescription: '重要信息集中呈现，日常使用与账户安全都更容易管理。', fastTitle: '高速节点接入', fastDescription: '按套餐使用对应节点组，节点更新后只需刷新订阅即可同步。', privacyTitle: '专属订阅保护', privacyDescription: '订阅令牌支持轮换与吊销，链接泄露后可及时替换。', simpleTitle: '多平台轻松使用', simpleDescription: '各平台均提供站内下载与清晰的安装、导入步骤。', greeting: '你好', verified: '已验证', status: '状态', active: '运行中', totalTraffic: '总流量', trafficReset: '流量将在当前周期结束时重置', importToClient: '导入订阅到 VPN 客户端', subscriptionLinkLabel: '订阅链接（专属链接，请勿泄露）', securityNote: '为保障账户安全，此链接仅供本人使用，请勿分享或公开。', clientStepDescription: '下载并安装支持的客户端', importStepDescription: '复制订阅链接并导入客户端', connectStepDescription: '选择节点，一键连接即可使用', allGuides: '查看全部教程', billingNotice: '所有金额以人民币结算，支付成功后自动开通。', recommended: '推荐', traffic: '流量', devices: '台设备', automaticActivation: '支付成功后自动开通', guidesDescription: '按设备下载客户端，再查看对应的订阅导入教程。', clientsDescription: '安装包由本站直接提供，无需跳转到其他网站。', signInForTickets: '登录后可以提交和查看工单', emptyTickets: '暂无工单', deviceLimit: '设备上限', gmailOnly: '仅支持使用 Gmail 邮箱验证码注册', inviteOptional: '邀请码（可选）', alipayTitle: '支付宝扫码支付', paymentTitle: '在线支付', choosePaymentMethod: '选择支付方式', choosePaymentMethodDescription: '请选择管理员已启用的一种支付方式继续付款。', noPaymentMethods: '当前没有可用的支付方式', orderLabel: '订单', paymentValidUntil: '二维码有效期至', confirmDemoPayment: '确认演示支付', subscriptionQRTitle: '订阅二维码', subscriptionQRWarning: '请勿分享此二维码；如有泄露，请立即轮换订阅链接。', orderNumber: '订单号', amount: '金额', createdAt: '创建时间', emptyOrders: '暂无订单', registrationSuccess: '注册成功', loginSuccess: '登录成功', codeSent: '验证码已发送', demoPaymentSuccess: '演示支付已确认，正在自动开通。', linkCopied: '订阅链接已复制', ticketSubmitted: '工单已提交', quota: '总额度', newPassword: '设置新密码', confirmPassword: '再次输入新密码', passwordRule: '密码至少 10 个字符，并包含大写字母、小写字母和数字；无需输入旧密码。', resetDescription: '验证 Gmail 邮箱后直接设置新密码，不需要提供原密码。', resetSuccess: '密码已重置，请使用新密码登录。', passwordMismatch: '两次输入的密码不一致', billingPeriod: '计费周期', couponCode: '兑换码（可选）', rotateLink: '轮换订阅链接', rotateConfirm: '旧订阅链接会立即失效，确定继续吗？', linkRotated: '新的订阅链接已生成', continuePayment: '继续支付', cancelOrder: '取消订单', orderCancelled: '订单已取消', ticketConversation: '工单对话', reply: '回复', accountSecurity: '账户与安全', sessions: '登录会话', currentSession: '当前会话', revokeSession: '退出该会话', balance: '账户余额', redeemGiftCard: '兑换礼品卡', giftCardCode: '礼品卡兑换码', redeemedSuccess: '礼品卡兑换成功' },
  'zh-TW': { ...en, home: '首頁', subscription: '我的訂閱', plans: '方案', guides: '使用教學', clients: '客戶端', tickets: '支援工單', signIn: '登入', signOut: '登出', currentPlan: '目前方案', used: '已使用', expires: '到期時間', daysLeft: '天後到期', manage: '管理訂閱', importSubscription: '匯入訂閱', importDescription: '複製專屬訂閱連結，或在支援的客戶端掃描 QR Code。', copyLink: '複製訂閱連結', showQR: '顯示 QR Code', quickStart: '快速開始', chooseClient: '取得官方客戶端', importProfile: '匯入訂閱連結', connect: '選擇節點並連線', officialDownload: '前往官方下載', buyNow: '立即選購', perMonth: '/ 月', noSubscription: '目前沒有有效訂閱', login: '登入', register: '註冊帳號', reset: '重設密碼', email: 'Gmail 信箱', password: '密碼', code: '驗證碼', sendCode: '傳送驗證碼', submit: '繼續', createTicket: '建立工單', subject: '主旨', body: '請描述您遇到的問題', orders: '訂單記錄', announcementDetails: '查看詳情', heroBadge: '安全穩定的連線服務', heroTitle: '一組訂閱，輕鬆連接所有裝置', heroDescription: '選擇方案、使用支付寶完成付款，再將專屬訂閱匯入支援的客戶端，幾分鐘即可開始使用。', browsePlans: '查看方案', readGuides: '閱讀教學', serviceTitle: '連線所需資訊，一站備齊', serviceDescription: '裝置教學、官方客戶端入口與訂閱管理集中呈現。', platformGuides: '個平台教學', subscriptionFormats: '種訂閱格式', selfService: '自助服務', benefitsTitle: '從購買到連線，流程簡單清楚', benefitsDescription: '重要資訊集中呈現，日常使用與帳戶安全更容易管理。', fastTitle: '高速節點接入', fastDescription: '依方案使用對應節點群組，節點更新後重新整理訂閱即可同步。', privacyTitle: '專屬訂閱保護', privacyDescription: '訂閱權杖支援輪換與撤銷，連結外洩後可立即更換。', simpleTitle: '跨平台輕鬆使用', simpleDescription: '依裝置查看專屬教學，僅前往官方商店或 GitHub 發布頁。', greeting: '您好', verified: '已驗證', status: '狀態', active: '使用中', totalTraffic: '總流量', trafficReset: '流量會在目前週期結束時重設', importToClient: '將訂閱匯入 VPN 客戶端', subscriptionLinkLabel: '訂閱連結（專屬連結，請勿外洩）', securityNote: '為保障帳戶安全，此連結僅供本人使用，請勿分享或公開。', clientStepDescription: '下載並安裝支援的官方客戶端', importStepDescription: '複製訂閱連結並匯入客戶端', connectStepDescription: '選擇節點，一鍵連線即可使用', allGuides: '查看全部教學', billingNotice: '所有金額以人民幣結算，付款成功後自動開通。', recommended: '推薦', traffic: '流量', devices: '部裝置', automaticActivation: '付款成功後自動開通', guidesDescription: '依裝置選擇教學，幾分鐘內完成訂閱匯入。', clientsDescription: '僅前往官方商店或 GitHub 發布頁，不重新封裝客戶端。', signInForTickets: '登入後即可提交及查看工單', emptyTickets: '目前沒有工單', deviceLimit: '裝置上限', gmailOnly: '僅支援使用 Gmail 驗證碼註冊', inviteOptional: '邀請碼（選填）', alipayTitle: '支付寶掃碼付款', orderLabel: '訂單', paymentValidUntil: 'QR Code 有效期至', confirmDemoPayment: '確認示範付款', subscriptionQRTitle: '訂閱 QR Code', subscriptionQRWarning: '請勿分享此 QR Code；如有外洩，請立即輪換訂閱連結。', orderNumber: '訂單編號', amount: '金額', createdAt: '建立時間', emptyOrders: '目前沒有訂單', registrationSuccess: '註冊成功', loginSuccess: '登入成功', codeSent: '驗證碼已傳送', demoPaymentSuccess: '示範付款已確認，系統正在自動開通。', linkCopied: '訂閱連結已複製', ticketSubmitted: '工單已送出', quota: '總額度' },
  'ja-JP': { ...en, home: 'ホーム', subscription: 'マイサブスクリプション', plans: '料金プラン', guides: '設定ガイド', clients: 'アプリ', tickets: 'サポート', signIn: 'ログイン', signOut: 'ログアウト', currentPlan: '現在のプラン', used: '使用済み', expires: '有効期限', daysLeft: '日後に期限切れ', manage: 'サブスクリプション管理', importSubscription: 'サブスクリプションを追加', importDescription: '専用リンクをコピーするか、対応アプリで QR コードを読み取ってください。', copyLink: 'リンクをコピー', showQR: 'QR コードを表示', quickStart: 'クイックスタート', chooseClient: '公式アプリを入手', importProfile: 'サブスクリプションを追加', connect: 'ノードを選んで接続', officialDownload: '公式ダウンロード', buyNow: 'プランを選択', perMonth: '/ 月', noSubscription: '有効なサブスクリプションはありません', login: 'ログイン', register: 'アカウント作成', reset: 'パスワードを再設定', email: 'Gmail アドレス', password: 'パスワード', code: '確認コード', sendCode: 'コードを送信', submit: '続ける', createTicket: 'チケットを作成', subject: '件名', body: 'お問い合わせ内容', orders: '注文履歴', announcementDetails: '詳細を見る', heroBadge: '安全で安定した接続サービス', heroTitle: '1つのサブスクリプションで、すべての端末をかんたん接続', heroDescription: 'プランを選び、Alipay で支払い、専用リンクを対応アプリに追加するだけです。', browsePlans: 'プランを見る', readGuides: 'ガイドを読む', serviceTitle: '接続に必要なものをひとつに', serviceDescription: '端末別ガイド、公式アプリ、サブスクリプション管理を一か所で利用できます。', platformGuides: '種類の端末ガイド', subscriptionFormats: '種類の形式', selfService: 'セルフサービス', benefitsTitle: '購入から接続まで、迷わない設計', benefitsDescription: '毎日の利用とアカウント保護に必要な情報を整理しています。', fastTitle: '高速ノードへ接続', fastDescription: 'プランのノードグループを利用し、変更時はサブスクリプションを更新できます。', privacyTitle: '専用リンクを保護', privacyDescription: 'リンクが漏えいした場合は、トークンをローテーションまたは無効化できます。', simpleTitle: '各端末でかんたん設定', simpleDescription: '端末別ガイドから公式ストアまたは GitHub のリリースのみを開きます。', greeting: 'こんにちは', verified: '確認済み', status: 'ステータス', active: '利用中', totalTraffic: '総通信量', trafficReset: '通信量は現在の期間終了時にリセットされます', importToClient: 'VPN アプリにサブスクリプションを追加', subscriptionLinkLabel: '専用リンク（共有しないでください）', securityNote: 'アカウント保護のため、このリンクを公開または共有しないでください。', clientStepDescription: '対応する公式アプリをダウンロードしてインストール', importStepDescription: 'リンクをコピーしてアプリに追加', connectStepDescription: 'ノードを選択して接続', allGuides: 'すべてのガイドを見る', billingNotice: '料金は人民元で決済され、支払い後に自動開通します。', recommended: 'おすすめ', traffic: '通信量', devices: '台', automaticActivation: '支払い後に自動開通', guidesDescription: '端末を選び、数分でサブスクリプションを追加できます。', clientsDescription: '公式ストアまたは GitHub のリリースのみを案内し、再配布は行いません。', signInForTickets: 'ログインするとチケットを作成・確認できます', emptyTickets: 'チケットはありません', deviceLimit: '端末上限', gmailOnly: 'Gmail の確認コードでのみ登録できます', inviteOptional: '招待コード（任意）', alipayTitle: 'Alipay で支払う', orderLabel: '注文', paymentValidUntil: 'QR コードの有効期限', confirmDemoPayment: 'デモ支払いを確定', subscriptionQRTitle: 'サブスクリプション QR コード', subscriptionQRWarning: 'QR コードを共有しないでください。漏えい時はリンクをすぐに更新してください。', orderNumber: '注文番号', amount: '金額', createdAt: '作成日時', emptyOrders: '注文はありません', registrationSuccess: 'アカウントを作成しました', loginSuccess: 'ログインしました', codeSent: '確認コードを送信しました', demoPaymentSuccess: 'デモ支払いを確認しました。開通処理を開始します。', linkCopied: 'リンクをコピーしました', ticketSubmitted: 'チケットを送信しました', quota: '上限' },
  'ru-RU': { ...en, home: 'Главная', subscription: 'Моя подписка', plans: 'Тарифы', guides: 'Инструкции', clients: 'Приложения', tickets: 'Поддержка', signIn: 'Войти', signOut: 'Выйти', currentPlan: 'Текущий тариф', used: 'Использовано', expires: 'Действует до', daysLeft: 'дн. осталось', manage: 'Управление подпиской', importSubscription: 'Импорт подписки', importDescription: 'Скопируйте личную ссылку или отсканируйте QR-код в поддерживаемом приложении.', copyLink: 'Скопировать ссылку', showQR: 'Показать QR-код', quickStart: 'Быстрый старт', chooseClient: 'Установить официальное приложение', importProfile: 'Импортировать подписку', connect: 'Выбрать узел и подключиться', officialDownload: 'Официальная загрузка', buyNow: 'Выбрать тариф', perMonth: '/ месяц', noSubscription: 'Активной подписки пока нет', login: 'Вход', register: 'Создать аккаунт', reset: 'Сбросить пароль', email: 'Адрес Gmail', password: 'Пароль', code: 'Код подтверждения', sendCode: 'Отправить код', submit: 'Продолжить', createTicket: 'Создать обращение', subject: 'Тема', body: 'Опишите проблему', orders: 'Заказы', announcementDetails: 'Подробнее', heroBadge: 'Безопасное и стабильное подключение', heroTitle: 'Одна подписка для простого подключения всех устройств', heroDescription: 'Выберите тариф, оплатите через Alipay и импортируйте личную подписку в поддерживаемое приложение.', browsePlans: 'Смотреть тарифы', readGuides: 'Открыть инструкции', serviceTitle: 'Всё необходимое для подключения', serviceDescription: 'Инструкции, официальные приложения и управление подпиской собраны в одном месте.', platformGuides: 'платформ с инструкциями', subscriptionFormats: 'формата подписки', selfService: 'самообслуживание', benefitsTitle: 'Понятный путь от оплаты до подключения', benefitsDescription: 'Важные данные для ежедневного использования и защиты аккаунта всегда под рукой.', fastTitle: 'Доступ к быстрым узлам', fastDescription: 'Используйте группу узлов тарифа и обновляйте подписку после изменений.', privacyTitle: 'Личная ссылка по умолчанию', privacyDescription: 'При утечке токен подписки можно заменить или отозвать.', simpleTitle: 'Просто на любом устройстве', simpleDescription: 'Используйте инструкции для платформ и переходите только в официальные магазины или GitHub.', greeting: 'Здравствуйте', verified: 'Подтверждено', status: 'Статус', active: 'Активна', totalTraffic: 'Общий трафик', trafficReset: 'Трафик сбрасывается по окончании текущего периода', importToClient: 'Импорт подписки в VPN-приложение', subscriptionLinkLabel: 'Личная ссылка — не передавайте её', securityNote: 'Не публикуйте и не передавайте эту ссылку другим людям.', clientStepDescription: 'Скачайте и установите поддерживаемое официальное приложение', importStepDescription: 'Скопируйте ссылку и импортируйте её в приложение', connectStepDescription: 'Выберите узел и подключитесь', allGuides: 'Все инструкции', billingNotice: 'Все цены указаны в юанях. Доступ активируется автоматически после оплаты.', recommended: 'Рекомендуем', traffic: 'трафика', devices: 'устройств', automaticActivation: 'Автоматическая активация после оплаты', guidesDescription: 'Выберите устройство и импортируйте подписку за несколько минут.', clientsDescription: 'Мы открываем только официальные магазины или релизы GitHub и не перепаковываем приложения.', signInForTickets: 'Войдите, чтобы создавать и просматривать обращения', emptyTickets: 'Обращений пока нет', deviceLimit: 'Лимит устройств', gmailOnly: 'Регистрация доступна только по коду, отправленному на Gmail', inviteOptional: 'Код приглашения (необязательно)', alipayTitle: 'Оплата через Alipay', orderLabel: 'Заказ', paymentValidUntil: 'QR-код действует до', confirmDemoPayment: 'Подтвердить демо-оплату', subscriptionQRTitle: 'QR-код подписки', subscriptionQRWarning: 'Не передавайте QR-код. При утечке немедленно замените ссылку.', orderNumber: 'Номер заказа', amount: 'Сумма', createdAt: 'Создан', emptyOrders: 'Заказов пока нет', registrationSuccess: 'Аккаунт создан', loginSuccess: 'Вход выполнен', codeSent: 'Код подтверждения отправлен', demoPaymentSuccess: 'Демо-оплата подтверждена. Начата активация.', linkCopied: 'Ссылка скопирована', ticketSubmitted: 'Обращение отправлено', quota: 'Лимит' },
  'vi-VN': { ...en, home: 'Trang chủ', subscription: 'Gói của tôi', plans: 'Gói dịch vụ', guides: 'Hướng dẫn', clients: 'Ứng dụng', tickets: 'Hỗ trợ', signIn: 'Đăng nhập', signOut: 'Đăng xuất', currentPlan: 'Gói hiện tại', used: 'Đã dùng', expires: 'Hết hạn', daysLeft: 'ngày còn lại', manage: 'Quản lý gói', importSubscription: 'Nhập đăng ký', importDescription: 'Sao chép liên kết riêng hoặc quét mã QR trong ứng dụng được hỗ trợ.', copyLink: 'Sao chép liên kết', showQR: 'Hiện mã QR', quickStart: 'Bắt đầu nhanh', chooseClient: 'Tải ứng dụng chính thức', importProfile: 'Nhập đăng ký', connect: 'Chọn máy chủ và kết nối', officialDownload: 'Tải chính thức', buyNow: 'Chọn gói', perMonth: '/ tháng', noSubscription: 'Chưa có gói đang hoạt động', login: 'Đăng nhập', register: 'Tạo tài khoản', reset: 'Đặt lại mật khẩu', email: 'Địa chỉ Gmail', password: 'Mật khẩu', code: 'Mã xác minh', sendCode: 'Gửi mã', submit: 'Tiếp tục', createTicket: 'Tạo yêu cầu', subject: 'Tiêu đề', body: 'Chúng tôi có thể giúp gì?', orders: 'Đơn hàng', announcementDetails: 'Xem chi tiết', heroBadge: 'Kết nối an toàn và ổn định', heroTitle: 'Một đăng ký để kết nối mọi thiết bị dễ dàng hơn', heroDescription: 'Chọn gói, thanh toán bằng Alipay và nhập liên kết riêng vào ứng dụng được hỗ trợ chỉ trong vài phút.', browsePlans: 'Xem các gói', readGuides: 'Đọc hướng dẫn', serviceTitle: 'Mọi thứ cần thiết để kết nối', serviceDescription: 'Hướng dẫn theo nền tảng, ứng dụng chính thức và quản lý đăng ký ở cùng một nơi.', platformGuides: 'nền tảng có hướng dẫn', subscriptionFormats: 'định dạng đăng ký', selfService: 'tự phục vụ', benefitsTitle: 'Trải nghiệm rõ ràng từ mua đến kết nối', benefitsDescription: 'Thông tin quan trọng cho sử dụng hằng ngày và bảo mật tài khoản luôn dễ tìm.', fastTitle: 'Truy cập máy chủ tốc độ cao', fastDescription: 'Dùng nhóm máy chủ của gói và cập nhật đăng ký khi danh sách thay đổi.', privacyTitle: 'Liên kết riêng tư', privacyDescription: 'Có thể đổi hoặc thu hồi mã đăng ký nếu liên kết bị lộ.', simpleTitle: 'Dễ dùng trên mọi thiết bị', simpleDescription: 'Làm theo hướng dẫn từng nền tảng và chỉ mở cửa hàng chính thức hoặc GitHub.', greeting: 'Xin chào', verified: 'Đã xác minh', status: 'Trạng thái', active: 'Đang hoạt động', totalTraffic: 'Tổng lưu lượng', trafficReset: 'Lưu lượng sẽ đặt lại khi kỳ hiện tại kết thúc', importToClient: 'Nhập đăng ký vào ứng dụng VPN', subscriptionLinkLabel: 'Liên kết riêng — không chia sẻ', securityNote: 'Để bảo vệ tài khoản, không công khai hoặc chia sẻ liên kết này.', clientStepDescription: 'Tải và cài ứng dụng chính thức được hỗ trợ', importStepDescription: 'Sao chép liên kết và nhập vào ứng dụng', connectStepDescription: 'Chọn máy chủ rồi kết nối', allGuides: 'Xem tất cả hướng dẫn', billingNotice: 'Mọi giá được tính bằng CNY. Dịch vụ tự kích hoạt sau khi thanh toán.', recommended: 'Đề xuất', traffic: 'lưu lượng', devices: 'thiết bị', automaticActivation: 'Tự động kích hoạt sau khi thanh toán', guidesDescription: 'Chọn thiết bị và hoàn tất nhập đăng ký trong vài phút.', clientsDescription: 'Chỉ mở cửa hàng chính thức hoặc bản phát hành GitHub; không đóng gói lại ứng dụng.', signInForTickets: 'Đăng nhập để tạo và xem yêu cầu hỗ trợ', emptyTickets: 'Chưa có yêu cầu hỗ trợ', deviceLimit: 'Giới hạn thiết bị', gmailOnly: 'Chỉ có thể đăng ký bằng mã xác minh gửi tới Gmail', inviteOptional: 'Mã mời (không bắt buộc)', alipayTitle: 'Thanh toán bằng Alipay', orderLabel: 'Đơn hàng', paymentValidUntil: 'Mã QR có hiệu lực đến', confirmDemoPayment: 'Xác nhận thanh toán thử', subscriptionQRTitle: 'Mã QR đăng ký', subscriptionQRWarning: 'Không chia sẻ mã QR. Hãy đổi liên kết ngay nếu bị lộ.', orderNumber: 'Mã đơn hàng', amount: 'Số tiền', createdAt: 'Ngày tạo', emptyOrders: 'Chưa có đơn hàng', registrationSuccess: 'Đã tạo tài khoản', loginSuccess: 'Đã đăng nhập', codeSent: 'Đã gửi mã xác minh', demoPaymentSuccess: 'Đã xác nhận thanh toán thử. Hệ thống đang kích hoạt.', linkCopied: 'Đã sao chép liên kết', ticketSubmitted: 'Đã gửi yêu cầu', quota: 'Hạn mức' },
  'es-ES': { ...en, home: 'Inicio', subscription: 'Mi suscripción', plans: 'Planes', guides: 'Guías de configuración', clients: 'Aplicaciones', tickets: 'Soporte', signIn: 'Iniciar sesión', signOut: 'Cerrar sesión', currentPlan: 'Plan actual', used: 'Usado', expires: 'Caduca', daysLeft: 'días restantes', manage: 'Gestionar suscripción', importSubscription: 'Importar suscripción', importDescription: 'Copia tu enlace privado o escanea el código QR en una aplicación compatible.', copyLink: 'Copiar enlace', showQR: 'Mostrar código QR', quickStart: 'Inicio rápido', chooseClient: 'Instalar una aplicación oficial', importProfile: 'Importar la suscripción', connect: 'Elegir un nodo y conectar', officialDownload: 'Descarga oficial', buyNow: 'Elegir plan', perMonth: '/ mes', noSubscription: 'Todavía no hay una suscripción activa', login: 'Iniciar sesión', register: 'Crear cuenta', reset: 'Restablecer contraseña', email: 'Dirección de Gmail', password: 'Contraseña', code: 'Código de verificación', sendCode: 'Enviar código', submit: 'Continuar', createTicket: 'Crear ticket', subject: 'Asunto', body: '¿Cómo podemos ayudarte?', orders: 'Pedidos', announcementDetails: 'Ver detalles', heroBadge: 'Acceso seguro y estable', heroTitle: 'Una suscripción para conectar todos tus dispositivos fácilmente', heroDescription: 'Elige un plan, paga con Alipay e importa tu enlace privado en una aplicación compatible en pocos minutos.', browsePlans: 'Ver planes', readGuides: 'Leer guías', serviceTitle: 'Todo lo necesario para conectarte', serviceDescription: 'Guías por plataforma, aplicaciones oficiales y gestión de la suscripción en un solo lugar.', platformGuides: 'plataformas con guía', subscriptionFormats: 'formatos de suscripción', selfService: 'acceso autoservicio', benefitsTitle: 'Una experiencia clara de la compra a la conexión', benefitsDescription: 'La información importante para el uso diario y la seguridad siempre está a mano.', fastTitle: 'Acceso a nodos rápidos', fastDescription: 'Usa el grupo de nodos del plan y actualiza la suscripción cuando haya cambios.', privacyTitle: 'Enlace privado por defecto', privacyDescription: 'Puedes rotar o revocar el token si el enlace queda expuesto.', simpleTitle: 'Fácil en cualquier dispositivo', simpleDescription: 'Sigue las guías por plataforma y abre solo tiendas oficiales o lanzamientos de GitHub.', greeting: 'Hola', verified: 'Verificado', status: 'Estado', active: 'Activa', totalTraffic: 'Tráfico total', trafficReset: 'El tráfico se restablece al terminar el periodo actual', importToClient: 'Importar la suscripción en tu aplicación VPN', subscriptionLinkLabel: 'Enlace privado — no lo compartas', securityNote: 'Para proteger tu cuenta, no publiques ni compartas este enlace.', clientStepDescription: 'Descarga e instala una aplicación oficial compatible', importStepDescription: 'Copia el enlace e impórtalo en la aplicación', connectStepDescription: 'Elige un nodo y conéctate', allGuides: 'Ver todas las guías', billingNotice: 'Todos los precios se cobran en CNY. El acceso se activa automáticamente tras el pago.', recommended: 'Recomendado', traffic: 'de tráfico', devices: 'dispositivos', automaticActivation: 'Activación automática después del pago', guidesDescription: 'Elige tu dispositivo e importa la suscripción en pocos minutos.', clientsDescription: 'Solo enlazamos tiendas oficiales o lanzamientos de GitHub; no reempaquetamos aplicaciones.', signInForTickets: 'Inicia sesión para crear y consultar tickets', emptyTickets: 'Aún no hay tickets', deviceLimit: 'Límite de dispositivos', gmailOnly: 'El registro solo está disponible mediante un código enviado a Gmail', inviteOptional: 'Código de invitación (opcional)', alipayTitle: 'Pagar con Alipay', orderLabel: 'Pedido', paymentValidUntil: 'Código QR válido hasta', confirmDemoPayment: 'Confirmar pago de demostración', subscriptionQRTitle: 'Código QR de suscripción', subscriptionQRWarning: 'No compartas este código QR. Cambia el enlace de inmediato si queda expuesto.', orderNumber: 'Número de pedido', amount: 'Importe', createdAt: 'Creado', emptyOrders: 'Aún no hay pedidos', registrationSuccess: 'Cuenta creada', loginSuccess: 'Sesión iniciada', codeSent: 'Código de verificación enviado', demoPaymentSuccess: 'Pago de demostración confirmado. La activación ha comenzado.', linkCopied: 'Enlace copiado', ticketSubmitted: 'Ticket enviado', quota: 'Cuota' },
  'id-ID': { ...en, home: 'Beranda', subscription: 'Langganan saya', plans: 'Paket', guides: 'Panduan pengaturan', clients: 'Aplikasi', tickets: 'Dukungan', signIn: 'Masuk', signOut: 'Keluar', currentPlan: 'Paket aktif', used: 'Terpakai', expires: 'Berakhir', daysLeft: 'hari tersisa', manage: 'Kelola langganan', importSubscription: 'Impor langganan', importDescription: 'Salin tautan pribadi atau pindai kode QR di aplikasi yang didukung.', copyLink: 'Salin tautan', showQR: 'Tampilkan kode QR', quickStart: 'Mulai cepat', chooseClient: 'Pasang aplikasi resmi', importProfile: 'Impor langganan', connect: 'Pilih node dan hubungkan', officialDownload: 'Unduhan resmi', buyNow: 'Pilih paket', perMonth: '/ bulan', noSubscription: 'Belum ada langganan aktif', login: 'Masuk', register: 'Buat akun', reset: 'Atur ulang kata sandi', email: 'Alamat Gmail', password: 'Kata sandi', code: 'Kode verifikasi', sendCode: 'Kirim kode', submit: 'Lanjut', createTicket: 'Buat tiket', subject: 'Subjek', body: 'Apa yang bisa kami bantu?', orders: 'Pesanan', announcementDetails: 'Lihat detail', heroBadge: 'Koneksi aman dan stabil', heroTitle: 'Satu langganan untuk menghubungkan semua perangkat dengan mudah', heroDescription: 'Pilih paket, bayar dengan Alipay, lalu impor tautan pribadi ke aplikasi yang didukung dalam beberapa menit.', browsePlans: 'Lihat paket', readGuides: 'Baca panduan', serviceTitle: 'Semua yang dibutuhkan untuk terhubung', serviceDescription: 'Panduan platform, aplikasi resmi, dan pengelolaan langganan dalam satu tempat.', platformGuides: 'panduan platform', subscriptionFormats: 'format langganan', selfService: 'akses mandiri', benefitsTitle: 'Pengalaman jelas dari pembelian hingga terhubung', benefitsDescription: 'Informasi penting untuk penggunaan harian dan keamanan akun mudah ditemukan.', fastTitle: 'Akses node cepat', fastDescription: 'Gunakan grup node dalam paket dan perbarui langganan saat ada perubahan.', privacyTitle: 'Tautan pribadi secara bawaan', privacyDescription: 'Token dapat diganti atau dicabut jika tautan terekspos.', simpleTitle: 'Mudah di setiap perangkat', simpleDescription: 'Ikuti panduan platform dan buka hanya toko resmi atau rilis GitHub.', greeting: 'Halo', verified: 'Terverifikasi', status: 'Status', active: 'Aktif', totalTraffic: 'Total lalu lintas', trafficReset: 'Lalu lintas diatur ulang saat periode saat ini berakhir', importToClient: 'Impor langganan ke aplikasi VPN', subscriptionLinkLabel: 'Tautan pribadi — jangan dibagikan', securityNote: 'Untuk melindungi akun, jangan publikasikan atau bagikan tautan ini.', clientStepDescription: 'Unduh dan pasang aplikasi resmi yang didukung', importStepDescription: 'Salin tautan dan impor ke aplikasi', connectStepDescription: 'Pilih node lalu hubungkan', allGuides: 'Lihat semua panduan', billingNotice: 'Semua harga ditagih dalam CNY. Akses aktif otomatis setelah pembayaran.', recommended: 'Direkomendasikan', traffic: 'lalu lintas', devices: 'perangkat', automaticActivation: 'Aktivasi otomatis setelah pembayaran', guidesDescription: 'Pilih perangkat dan selesaikan impor dalam beberapa menit.', clientsDescription: 'Hanya membuka toko resmi atau rilis GitHub; aplikasi tidak dikemas ulang.', signInForTickets: 'Masuk untuk membuat dan melihat tiket dukungan', emptyTickets: 'Belum ada tiket dukungan', deviceLimit: 'Batas perangkat', gmailOnly: 'Pendaftaran hanya tersedia dengan kode verifikasi Gmail', inviteOptional: 'Kode undangan (opsional)', alipayTitle: 'Bayar dengan Alipay', orderLabel: 'Pesanan', paymentValidUntil: 'Kode QR berlaku hingga', confirmDemoPayment: 'Konfirmasi pembayaran demo', subscriptionQRTitle: 'Kode QR langganan', subscriptionQRWarning: 'Jangan bagikan kode QR. Ganti tautan segera jika terekspos.', orderNumber: 'Nomor pesanan', amount: 'Jumlah', createdAt: 'Dibuat', emptyOrders: 'Belum ada pesanan', registrationSuccess: 'Akun dibuat', loginSuccess: 'Berhasil masuk', codeSent: 'Kode verifikasi dikirim', demoPaymentSuccess: 'Pembayaran demo dikonfirmasi. Aktivasi dimulai.', linkCopied: 'Tautan disalin', ticketSubmitted: 'Tiket dikirim', quota: 'Kuota' },
  'uk-UA': { ...en, home: 'Головна', subscription: 'Моя підписка', plans: 'Тарифи', guides: 'Інструкції', clients: 'Застосунки', tickets: 'Підтримка', signIn: 'Увійти', signOut: 'Вийти', currentPlan: 'Поточний тариф', used: 'Використано', expires: 'Діє до', daysLeft: 'днів залишилося', manage: 'Керувати підпискою', importSubscription: 'Імпортувати підписку', importDescription: 'Скопіюйте приватне посилання або відскануйте QR-код у підтримуваному застосунку.', copyLink: 'Скопіювати посилання', showQR: 'Показати QR-код', quickStart: 'Швидкий старт', chooseClient: 'Установити офіційний застосунок', importProfile: 'Імпортувати підписку', connect: 'Обрати вузол і підключитися', officialDownload: 'Офіційне завантаження', buyNow: 'Обрати тариф', perMonth: '/ місяць', noSubscription: 'Активної підписки ще немає', login: 'Вхід', register: 'Створити обліковий запис', reset: 'Скинути пароль', email: 'Адреса Gmail', password: 'Пароль', code: 'Код підтвердження', sendCode: 'Надіслати код', submit: 'Продовжити', createTicket: 'Створити звернення', subject: 'Тема', body: 'Як ми можемо допомогти?', orders: 'Замовлення', announcementDetails: 'Докладніше', heroBadge: 'Безпечне та стабільне підключення', heroTitle: 'Одна підписка для простого підключення всіх пристроїв', heroDescription: 'Оберіть тариф, сплатіть через Alipay та імпортуйте приватне посилання у підтримуваний застосунок.', browsePlans: 'Переглянути тарифи', readGuides: 'Відкрити інструкції', serviceTitle: 'Усе необхідне для підключення', serviceDescription: 'Інструкції, офіційні застосунки та керування підпискою зібрані в одному місці.', platformGuides: 'платформ з інструкціями', subscriptionFormats: 'формати підписки', selfService: 'самообслуговування', benefitsTitle: 'Зрозумілий шлях від оплати до підключення', benefitsDescription: 'Важливі дані для щоденного використання та захисту облікового запису завжди поруч.', fastTitle: 'Доступ до швидких вузлів', fastDescription: 'Використовуйте групу вузлів тарифу й оновлюйте підписку після змін.', privacyTitle: 'Приватне посилання', privacyDescription: 'У разі витоку токен підписки можна замінити або відкликати.', simpleTitle: 'Просто на кожному пристрої', simpleDescription: 'Дотримуйтеся інструкцій платформи й відкривайте лише офіційні магазини або GitHub.', greeting: 'Вітаємо', verified: 'Підтверджено', status: 'Статус', active: 'Активна', totalTraffic: 'Загальний трафік', trafficReset: 'Трафік скидається після завершення поточного періоду', importToClient: 'Імпортувати підписку у VPN-застосунок', subscriptionLinkLabel: 'Приватне посилання — не поширюйте його', securityNote: 'Задля безпеки не публікуйте й не передавайте це посилання.', clientStepDescription: 'Завантажте й установіть підтримуваний офіційний застосунок', importStepDescription: 'Скопіюйте посилання та імпортуйте його', connectStepDescription: 'Оберіть вузол і підключіться', allGuides: 'Усі інструкції', billingNotice: 'Усі ціни вказані в юанях. Доступ активується автоматично після оплати.', recommended: 'Рекомендуємо', traffic: 'трафіку', devices: 'пристроїв', automaticActivation: 'Автоматична активація після оплати', guidesDescription: 'Оберіть пристрій та імпортуйте підписку за кілька хвилин.', clientsDescription: 'Ми відкриваємо лише офіційні магазини або релізи GitHub і не перепаковуємо застосунки.', signInForTickets: 'Увійдіть, щоб створювати й переглядати звернення', emptyTickets: 'Звернень ще немає', deviceLimit: 'Ліміт пристроїв', gmailOnly: 'Реєстрація доступна лише за кодом із Gmail', inviteOptional: 'Код запрошення (необов’язково)', alipayTitle: 'Сплатити через Alipay', orderLabel: 'Замовлення', paymentValidUntil: 'QR-код дійсний до', confirmDemoPayment: 'Підтвердити демооплату', subscriptionQRTitle: 'QR-код підписки', subscriptionQRWarning: 'Не поширюйте QR-код. У разі витоку негайно замініть посилання.', orderNumber: 'Номер замовлення', amount: 'Сума', createdAt: 'Створено', emptyOrders: 'Замовлень ще немає', registrationSuccess: 'Обліковий запис створено', loginSuccess: 'Вхід виконано', codeSent: 'Код підтвердження надіслано', demoPaymentSuccess: 'Демооплату підтверджено. Активацію розпочато.', linkCopied: 'Посилання скопійовано', ticketSubmitted: 'Звернення надіслано', quota: 'Ліміт' },
  'tr-TR': { ...en, home: 'Ana sayfa', subscription: 'Aboneliğim', plans: 'Paketler', guides: 'Kurulum rehberleri', clients: 'Uygulamalar', tickets: 'Destek', signIn: 'Giriş yap', signOut: 'Çıkış yap', currentPlan: 'Mevcut paket', used: 'Kullanılan', expires: 'Bitiş tarihi', daysLeft: 'gün kaldı', manage: 'Aboneliği yönet', importSubscription: 'Aboneliği içe aktar', importDescription: 'Özel bağlantınızı kopyalayın veya desteklenen bir uygulamada QR kodunu tarayın.', copyLink: 'Bağlantıyı kopyala', showQR: 'QR kodunu göster', quickStart: 'Hızlı başlangıç', chooseClient: 'Resmî uygulamayı yükle', importProfile: 'Aboneliği içe aktar', connect: 'Düğüm seçip bağlan', officialDownload: 'Resmî indirme', buyNow: 'Paket seç', perMonth: '/ ay', noSubscription: 'Henüz etkin abonelik yok', login: 'Giriş yap', register: 'Hesap oluştur', reset: 'Şifreyi sıfırla', email: 'Gmail adresi', password: 'Şifre', code: 'Doğrulama kodu', sendCode: 'Kod gönder', submit: 'Devam', createTicket: 'Destek talebi oluştur', subject: 'Konu', body: 'Size nasıl yardımcı olabiliriz?', orders: 'Siparişler', announcementDetails: 'Ayrıntıları gör', heroBadge: 'Güvenli ve kararlı bağlantı', heroTitle: 'Tüm cihazlar için tek ve kolay bir abonelik', heroDescription: 'Paket seçin, Alipay ile ödeyin ve özel bağlantınızı desteklenen uygulamaya birkaç dakikada aktarın.', browsePlans: 'Paketleri gör', readGuides: 'Rehberleri oku', serviceTitle: 'Bağlanmak için gereken her şey', serviceDescription: 'Platform rehberleri, resmî uygulamalar ve abonelik yönetimi tek yerde.', platformGuides: 'platform rehberi', subscriptionFormats: 'abonelik biçimi', selfService: 'kendi kendine hizmet', benefitsTitle: 'Satın almadan bağlantıya kadar net bir deneyim', benefitsDescription: 'Günlük kullanım ve hesap güvenliği için önemli bilgiler kolayca erişilebilir.', fastTitle: 'Hızlı düğümlere erişim', fastDescription: 'Paketinizin düğüm grubunu kullanın ve değişikliklerde aboneliği yenileyin.', privacyTitle: 'Varsayılan olarak özel', privacyDescription: 'Bağlantı açığa çıkarsa abonelik belirtecini değiştirebilir veya iptal edebilirsiniz.', simpleTitle: 'Her cihazda kolay', simpleDescription: 'Platform rehberlerini izleyin ve yalnızca resmî mağazaları veya GitHub sürümlerini açın.', greeting: 'Merhaba', verified: 'Doğrulandı', status: 'Durum', active: 'Etkin', totalTraffic: 'Toplam trafik', trafficReset: 'Trafik mevcut dönem bitince sıfırlanır', importToClient: 'Aboneliği VPN uygulamasına aktar', subscriptionLinkLabel: 'Özel bağlantı — paylaşmayın', securityNote: 'Hesap güvenliği için bu bağlantıyı yayımlamayın veya paylaşmayın.', clientStepDescription: 'Desteklenen resmî uygulamayı indirin ve yükleyin', importStepDescription: 'Bağlantıyı kopyalayıp uygulamaya aktarın', connectStepDescription: 'Düğüm seçip bağlanın', allGuides: 'Tüm rehberler', billingNotice: 'Tüm fiyatlar CNY üzerinden alınır. Ödeme sonrası erişim otomatik açılır.', recommended: 'Önerilen', traffic: 'trafik', devices: 'cihaz', automaticActivation: 'Ödeme sonrası otomatik etkinleştirme', guidesDescription: 'Cihazınızı seçin ve aboneliği birkaç dakikada içe aktarın.', clientsDescription: 'Yalnızca resmî mağazalara veya GitHub sürümlerine yönlendiririz; uygulamaları yeniden paketlemeyiz.', signInForTickets: 'Destek taleplerini oluşturmak ve görmek için giriş yapın', emptyTickets: 'Henüz destek talebi yok', deviceLimit: 'Cihaz sınırı', gmailOnly: 'Kayıt yalnızca Gmail doğrulama koduyla yapılabilir', inviteOptional: 'Davet kodu (isteğe bağlı)', alipayTitle: 'Alipay ile öde', orderLabel: 'Sipariş', paymentValidUntil: 'QR kodu şu tarihe kadar geçerli', confirmDemoPayment: 'Demo ödemeyi onayla', subscriptionQRTitle: 'Abonelik QR kodu', subscriptionQRWarning: 'QR kodunu paylaşmayın. Açığa çıkarsa bağlantıyı hemen değiştirin.', orderNumber: 'Sipariş numarası', amount: 'Tutar', createdAt: 'Oluşturuldu', emptyOrders: 'Henüz sipariş yok', registrationSuccess: 'Hesap oluşturuldu', loginSuccess: 'Giriş yapıldı', codeSent: 'Doğrulama kodu gönderildi', demoPaymentSuccess: 'Demo ödeme onaylandı. Etkinleştirme başladı.', linkCopied: 'Bağlantı kopyalandı', ticketSubmitted: 'Destek talebi gönderildi', quota: 'Kota' },
  'pt-BR': { ...en, home: 'Início', subscription: 'Minha assinatura', plans: 'Planos', guides: 'Guias de configuração', clients: 'Aplicativos', tickets: 'Suporte', signIn: 'Entrar', signOut: 'Sair', currentPlan: 'Plano atual', used: 'Usado', expires: 'Expira em', daysLeft: 'dias restantes', manage: 'Gerenciar assinatura', importSubscription: 'Importar assinatura', importDescription: 'Copie seu link privado ou escaneie o código QR em um aplicativo compatível.', copyLink: 'Copiar link', showQR: 'Mostrar código QR', quickStart: 'Início rápido', chooseClient: 'Instalar aplicativo oficial', importProfile: 'Importar assinatura', connect: 'Escolher um nó e conectar', officialDownload: 'Download oficial', buyNow: 'Escolher plano', perMonth: '/ mês', noSubscription: 'Ainda não há assinatura ativa', login: 'Entrar', register: 'Criar conta', reset: 'Redefinir senha', email: 'Endereço do Gmail', password: 'Senha', code: 'Código de verificação', sendCode: 'Enviar código', submit: 'Continuar', createTicket: 'Criar chamado', subject: 'Assunto', body: 'Como podemos ajudar?', orders: 'Pedidos', announcementDetails: 'Ver detalhes', heroBadge: 'Acesso seguro e estável', heroTitle: 'Uma assinatura para conectar todos os seus dispositivos com facilidade', heroDescription: 'Escolha um plano, pague com Alipay e importe seu link privado em um aplicativo compatível em poucos minutos.', browsePlans: 'Ver planos', readGuides: 'Ler guias', serviceTitle: 'Tudo o que você precisa para se conectar', serviceDescription: 'Guias por plataforma, aplicativos oficiais e gerenciamento da assinatura em um só lugar.', platformGuides: 'plataformas com guia', subscriptionFormats: 'formatos de assinatura', selfService: 'acesso por autoatendimento', benefitsTitle: 'Uma experiência clara da compra à conexão', benefitsDescription: 'As informações importantes para uso diário e segurança ficam sempre à mão.', fastTitle: 'Acesso a nós rápidos', fastDescription: 'Use o grupo de nós do plano e atualize a assinatura quando houver mudanças.', privacyTitle: 'Link privado por padrão', privacyDescription: 'Você pode trocar ou revogar o token caso o link seja exposto.', simpleTitle: 'Fácil em qualquer dispositivo', simpleDescription: 'Siga os guias por plataforma e abra apenas lojas oficiais ou versões do GitHub.', greeting: 'Olá', verified: 'Verificado', status: 'Status', active: 'Ativa', totalTraffic: 'Tráfego total', trafficReset: 'O tráfego é redefinido ao fim do período atual', importToClient: 'Importar assinatura no aplicativo VPN', subscriptionLinkLabel: 'Link privado — não compartilhe', securityNote: 'Para proteger sua conta, não publique nem compartilhe este link.', clientStepDescription: 'Baixe e instale um aplicativo oficial compatível', importStepDescription: 'Copie o link e importe no aplicativo', connectStepDescription: 'Escolha um nó e conecte', allGuides: 'Ver todos os guias', billingNotice: 'Todos os preços são cobrados em CNY. O acesso é ativado automaticamente após o pagamento.', recommended: 'Recomendado', traffic: 'de tráfego', devices: 'dispositivos', automaticActivation: 'Ativação automática após o pagamento', guidesDescription: 'Escolha o dispositivo e importe a assinatura em poucos minutos.', clientsDescription: 'Abrimos apenas lojas oficiais ou versões do GitHub; os aplicativos não são reempacotados.', signInForTickets: 'Entre para criar e consultar chamados', emptyTickets: 'Ainda não há chamados', deviceLimit: 'Limite de dispositivos', gmailOnly: 'O cadastro está disponível apenas com código de verificação do Gmail', inviteOptional: 'Código de convite (opcional)', alipayTitle: 'Pagar com Alipay', orderLabel: 'Pedido', paymentValidUntil: 'Código QR válido até', confirmDemoPayment: 'Confirmar pagamento de demonstração', subscriptionQRTitle: 'Código QR da assinatura', subscriptionQRWarning: 'Não compartilhe este código QR. Troque o link imediatamente se ele for exposto.', orderNumber: 'Número do pedido', amount: 'Valor', createdAt: 'Criado em', emptyOrders: 'Ainda não há pedidos', registrationSuccess: 'Conta criada', loginSuccess: 'Login realizado', codeSent: 'Código de verificação enviado', demoPaymentSuccess: 'Pagamento de demonstração confirmado. A ativação foi iniciada.', linkCopied: 'Link copiado', ticketSubmitted: 'Chamado enviado', quota: 'Cota' },
  'ar-EG': { ...en, home: 'الرئيسية', subscription: 'اشتراكي', plans: 'الباقات', guides: 'دليل الإعداد', clients: 'التطبيقات', tickets: 'الدعم', signIn: 'تسجيل الدخول', signOut: 'تسجيل الخروج', currentPlan: 'الباقة الحالية', used: 'المستخدم', expires: 'ينتهي في', daysLeft: 'يوم متبقٍ', manage: 'إدارة الاشتراك', importSubscription: 'استيراد الاشتراك', importDescription: 'انسخ رابطك الخاص أو امسح رمز QR داخل تطبيق مدعوم.', copyLink: 'نسخ رابط الاشتراك', showQR: 'عرض رمز QR', quickStart: 'بدء سريع', chooseClient: 'تثبيت تطبيق رسمي', importProfile: 'استيراد الاشتراك', connect: 'اختر عقدة واتصل', officialDownload: 'تنزيل رسمي', buyNow: 'اختيار الباقة', perMonth: '/ شهر', noSubscription: 'لا يوجد اشتراك نشط حتى الآن', login: 'تسجيل الدخول', register: 'إنشاء حساب', reset: 'إعادة تعيين كلمة المرور', email: 'عنوان Gmail', password: 'كلمة المرور', code: 'رمز التحقق', sendCode: 'إرسال الرمز', submit: 'متابعة', createTicket: 'إنشاء تذكرة', subject: 'الموضوع', body: 'كيف يمكننا مساعدتك؟', orders: 'الطلبات', announcementDetails: 'عرض التفاصيل', heroBadge: 'اتصال آمن ومستقر', heroTitle: 'اشتراك واحد لاتصال أسهل على جميع أجهزتك', heroDescription: 'اختر باقة وادفع عبر Alipay ثم استورد رابطك الخاص في تطبيق مدعوم خلال دقائق.', browsePlans: 'عرض الباقات', readGuides: 'قراءة الأدلة', serviceTitle: 'كل ما تحتاجه للاتصال', serviceDescription: 'أدلة المنصات والتطبيقات الرسمية وإدارة الاشتراك في مكان واحد.', platformGuides: 'أدلة للمنصات', subscriptionFormats: 'صيغ اشتراك', selfService: 'خدمة ذاتية', benefitsTitle: 'تجربة واضحة من الشراء حتى الاتصال', benefitsDescription: 'المعلومات المهمة للاستخدام اليومي وأمان الحساب متاحة بسهولة.', fastTitle: 'الوصول إلى عقد سريعة', fastDescription: 'استخدم مجموعة العقد ضمن باقتك وحدّث الاشتراك عند تغيير العقد.', privacyTitle: 'رابط خاص افتراضيًا', privacyDescription: 'يمكنك تدوير رمز الاشتراك أو إلغاؤه إذا انكشف الرابط.', simpleTitle: 'سهل على كل جهاز', simpleDescription: 'اتبع دليل منصتك وافتح المتاجر الرسمية أو إصدارات GitHub فقط.', greeting: 'مرحبًا', verified: 'تم التحقق', status: 'الحالة', active: 'نشط', totalTraffic: 'إجمالي البيانات', trafficReset: 'تُعاد تهيئة البيانات عند انتهاء الفترة الحالية', importToClient: 'استيراد الاشتراك إلى تطبيق VPN', subscriptionLinkLabel: 'رابط خاص — لا تشاركه', securityNote: 'لحماية حسابك، لا تنشر هذا الرابط ولا تشاركه.', clientStepDescription: 'نزّل وثبّت تطبيقًا رسميًا مدعومًا', importStepDescription: 'انسخ الرابط واستورده داخل التطبيق', connectStepDescription: 'اختر عقدة واتصل', allGuides: 'عرض جميع الأدلة', billingNotice: 'تُحصّل جميع الأسعار باليوان الصيني ويُفعّل الوصول تلقائيًا بعد الدفع.', recommended: 'موصى بها', traffic: 'بيانات', devices: 'أجهزة', automaticActivation: 'تفعيل تلقائي بعد الدفع', guidesDescription: 'اختر جهازك واستورد الاشتراك خلال دقائق.', clientsDescription: 'نفتح المتاجر الرسمية أو إصدارات GitHub فقط ولا نعيد حزم التطبيقات.', signInForTickets: 'سجّل الدخول لإنشاء تذاكر الدعم وعرضها', emptyTickets: 'لا توجد تذاكر دعم بعد', deviceLimit: 'حد الأجهزة', gmailOnly: 'التسجيل متاح فقط باستخدام رمز تحقق يُرسل إلى Gmail', inviteOptional: 'رمز الدعوة (اختياري)', alipayTitle: 'الدفع عبر Alipay', orderLabel: 'الطلب', paymentValidUntil: 'رمز QR صالح حتى', confirmDemoPayment: 'تأكيد الدفع التجريبي', subscriptionQRTitle: 'رمز QR للاشتراك', subscriptionQRWarning: 'لا تشارك رمز QR. غيّر رابط الاشتراك فورًا إذا انكشف.', orderNumber: 'رقم الطلب', amount: 'المبلغ', createdAt: 'تاريخ الإنشاء', emptyOrders: 'لا توجد طلبات بعد', registrationSuccess: 'تم إنشاء الحساب', loginSuccess: 'تم تسجيل الدخول', codeSent: 'تم إرسال رمز التحقق', demoPaymentSuccess: 'تم تأكيد الدفع التجريبي وبدأ التفعيل.', linkCopied: 'تم نسخ رابط الاشتراك', ticketSubmitted: 'تم إرسال التذكرة', quota: 'السعة' },
  'fa-IR': { ...en, home: 'خانه', subscription: 'اشتراک من', plans: 'طرح‌ها', guides: 'راهنمای راه‌اندازی', clients: 'برنامه‌ها', tickets: 'پشتیبانی', signIn: 'ورود', signOut: 'خروج', currentPlan: 'طرح فعلی', used: 'مصرف‌شده', expires: 'تاریخ انقضا', daysLeft: 'روز باقی‌مانده', manage: 'مدیریت اشتراک', importSubscription: 'افزودن اشتراک', importDescription: 'پیوند خصوصی را کپی کنید یا کد QR را در یک برنامه پشتیبانی‌شده اسکن کنید.', copyLink: 'کپی پیوند اشتراک', showQR: 'نمایش کد QR', quickStart: 'شروع سریع', chooseClient: 'نصب برنامه رسمی', importProfile: 'افزودن اشتراک', connect: 'انتخاب گره و اتصال', officialDownload: 'دانلود رسمی', buyNow: 'انتخاب طرح', perMonth: '/ ماه', noSubscription: 'هنوز اشتراک فعالی ندارید', login: 'ورود', register: 'ساخت حساب', reset: 'بازنشانی رمز عبور', email: 'نشانی Gmail', password: 'رمز عبور', code: 'کد تأیید', sendCode: 'ارسال کد', submit: 'ادامه', createTicket: 'ایجاد تیکت', subject: 'موضوع', body: 'چطور می‌توانیم کمک کنیم؟', orders: 'سفارش‌ها', announcementDetails: 'مشاهده جزئیات', heroBadge: 'اتصال امن و پایدار', heroTitle: 'یک اشتراک برای اتصال آسان‌تر همه دستگاه‌ها', heroDescription: 'طرح را انتخاب کنید، با Alipay بپردازید و پیوند خصوصی را در چند دقیقه وارد برنامه کنید.', browsePlans: 'مشاهده طرح‌ها', readGuides: 'خواندن راهنماها', serviceTitle: 'همه چیز برای اتصال', serviceDescription: 'راهنمای پلتفرم‌ها، برنامه‌های رسمی و مدیریت اشتراک در یک مکان.', platformGuides: 'راهنمای پلتفرم', subscriptionFormats: 'قالب اشتراک', selfService: 'دسترسی سلف‌سرویس', benefitsTitle: 'مسیر روشن از خرید تا اتصال', benefitsDescription: 'اطلاعات مهم استفاده روزانه و امنیت حساب همیشه در دسترس است.', fastTitle: 'دسترسی به گره‌های سریع', fastDescription: 'از گروه گره طرح استفاده کنید و هنگام تغییر گره‌ها اشتراک را تازه کنید.', privacyTitle: 'پیوند خصوصی به‌صورت پیش‌فرض', privacyDescription: 'اگر پیوند افشا شد می‌توانید توکن اشتراک را تعویض یا لغو کنید.', simpleTitle: 'ساده روی هر دستگاه', simpleDescription: 'راهنمای پلتفرم را دنبال کنید و فقط فروشگاه رسمی یا انتشار GitHub را باز کنید.', greeting: 'سلام', verified: 'تأییدشده', status: 'وضعیت', active: 'فعال', totalTraffic: 'کل ترافیک', trafficReset: 'ترافیک در پایان دوره فعلی بازنشانی می‌شود', importToClient: 'افزودن اشتراک به برنامه VPN', subscriptionLinkLabel: 'پیوند خصوصی — آن را به اشتراک نگذارید', securityNote: 'برای امنیت حساب، این پیوند را منتشر یا به اشتراک نگذارید.', clientStepDescription: 'یک برنامه رسمی پشتیبانی‌شده را دانلود و نصب کنید', importStepDescription: 'پیوند را کپی و در برنامه وارد کنید', connectStepDescription: 'گره را انتخاب و متصل شوید', allGuides: 'مشاهده همه راهنماها', billingNotice: 'همه قیمت‌ها به یوان چین دریافت می‌شوند و دسترسی پس از پرداخت خودکار فعال می‌شود.', recommended: 'پیشنهادی', traffic: 'ترافیک', devices: 'دستگاه', automaticActivation: 'فعال‌سازی خودکار پس از پرداخت', guidesDescription: 'دستگاه را انتخاب و اشتراک را در چند دقیقه وارد کنید.', clientsDescription: 'فقط فروشگاه رسمی یا انتشار GitHub باز می‌شود و برنامه‌ها بازبسته‌بندی نمی‌شوند.', signInForTickets: 'برای ایجاد و مشاهده تیکت‌های پشتیبانی وارد شوید', emptyTickets: 'هنوز تیکتی وجود ندارد', deviceLimit: 'سقف دستگاه', gmailOnly: 'ثبت‌نام فقط با کد تأیید ارسالی به Gmail امکان‌پذیر است', inviteOptional: 'کد دعوت (اختیاری)', alipayTitle: 'پرداخت با Alipay', orderLabel: 'سفارش', paymentValidUntil: 'کد QR معتبر تا', confirmDemoPayment: 'تأیید پرداخت آزمایشی', subscriptionQRTitle: 'کد QR اشتراک', subscriptionQRWarning: 'کد QR را به اشتراک نگذارید. در صورت افشا، پیوند را فوراً عوض کنید.', orderNumber: 'شماره سفارش', amount: 'مبلغ', createdAt: 'زمان ایجاد', emptyOrders: 'هنوز سفارشی وجود ندارد', registrationSuccess: 'حساب ساخته شد', loginSuccess: 'وارد شدید', codeSent: 'کد تأیید ارسال شد', demoPaymentSuccess: 'پرداخت آزمایشی تأیید شد و فعال‌سازی آغاز شده است.', linkCopied: 'پیوند اشتراک کپی شد', ticketSubmitted: 'تیکت ارسال شد', quota: 'سهمیه' },
};

Object.assign(portalCopies['en-US'], { guides: 'Documentation', guidesDescription: 'Choose an official app for your device, then follow the matching import guide.' });
Object.assign(portalCopies['zh-CN'], { guides: '使用文档', guidesDescription: '先按设备选择官方客户端，再查看对应的订阅导入教程。', acceptTermsPrefix: '我已阅读并同意', termsOfService: '《使用条款》', termsRequired: '请先阅读并勾选同意使用条款', readTerms: '查看使用条款', acceptTermsAction: '同意并继续' });
Object.assign(portalCopies['zh-TW'], { guides: '使用說明', guidesDescription: '先依裝置選擇官方客戶端，再查看對應的訂閱匯入教學。', acceptTermsPrefix: '我已閱讀並同意', termsOfService: '《使用條款》', termsRequired: '請先閱讀並勾選同意使用條款', readTerms: '查看使用條款', acceptTermsAction: '同意並繼續' });
Object.assign(portalCopies['ja-JP'], { guides: 'ご利用ガイド', guidesDescription: '端末に合う公式アプリを選び、対応するインポート手順を確認してください。' });
Object.assign(portalCopies['ru-RU'], { guides: 'Документация', guidesDescription: 'Выберите официальное приложение для устройства, затем откройте подходящую инструкцию импорта.' });
Object.assign(portalCopies['vi-VN'], { guides: 'Tài liệu sử dụng', guidesDescription: 'Chọn ứng dụng chính thức cho thiết bị, sau đó làm theo hướng dẫn nhập tương ứng.' });
Object.assign(portalCopies['es-ES'], { guides: 'Documentación', guidesDescription: 'Elige la aplicación oficial para tu dispositivo y sigue la guía de importación correspondiente.' });
Object.assign(portalCopies['id-ID'], { guides: 'Dokumentasi', guidesDescription: 'Pilih aplikasi resmi untuk perangkat Anda, lalu ikuti panduan impor yang sesuai.' });
Object.assign(portalCopies['uk-UA'], { guides: 'Документація', guidesDescription: 'Виберіть офіційний застосунок для пристрою, а потім відкрийте відповідну інструкцію імпорту.' });
Object.assign(portalCopies['tr-TR'], { guides: 'Kullanım belgeleri', guidesDescription: 'Cihazınız için resmi uygulamayı seçin, ardından uygun içe aktarma kılavuzunu izleyin.' });
Object.assign(portalCopies['pt-BR'], { guides: 'Documentação', guidesDescription: 'Escolha o aplicativo oficial para seu dispositivo e siga o guia de importação correspondente.' });
Object.assign(portalCopies['ar-EG'], { guides: 'دليل الاستخدام', guidesDescription: 'اختر التطبيق الرسمي المناسب لجهازك ثم اتبع دليل الاستيراد المطابق.' });
Object.assign(portalCopies['ar-EG'], {
  accountCenter: 'المركز الشخصي',
  accountOverview: 'نظرة عامة',
  invitationRewards: 'مكافآت الدعوة',
  invitationCode: 'رمز الدعوة',
  invitationLink: 'رابط الدعوة الشخصي',
  copyInvitation: 'نسخ رابط الدعوة',
  inviteFriends: 'دعوة الأصدقاء',
  invitationDescription: 'شارك رابط دعوتك الشخصي. تُضاف المكافآت إلى رصيد حسابك وفق سياسة الموقع الحالية.',
  accountSecurity: 'أمان الحساب',
  balance: 'رصيد الحساب',
  invitedUsers: 'الأصدقاء المدعوون',
  commissionRate: 'نسبة المكافأة',
  pendingCommission: 'المكافآت المعلقة',
  earnedCommission: 'المكافآت المكتسبة',
  commissionBalanceHint: 'لا يمكن سحب رصيد الحساب، ويمكن استخدامه فقط لشراء الباقات أو تجديدها أو ترقيتها.',
  firstPaymentRule: 'تُحسب المكافأة على أول دفعة ناجحة للمستخدم المدعو.',
  inviteCodePermanent: 'رمز دعوتك لا تنتهي صلاحيته.',
  redeemGiftCard: 'استرداد بطاقة هدية',
  giftCardCode: 'رمز بطاقة الهدية',
  subscription: 'اشتراكي',
  currentPlan: 'الباقة الحالية',
  status: 'الحالة',
  active: 'نشط',
  used: 'المستخدم',
  totalTraffic: 'إجمالي البيانات',
  trafficReset: 'تُعاد تهيئة البيانات عند انتهاء الدورة الحالية',
});
Object.assign(portalCopies['fa-IR'], { guides: 'راهنمای استفاده', guidesDescription: 'برنامه رسمی مناسب دستگاه را انتخاب کنید و سپس راهنمای ورود مرتبط را دنبال کنید.' });

Object.assign(portalCopies['zh-CN'], { changePassword: '修改密码', currentPassword: '当前密码', passwordRule: '密码至少 10 个字符，并包含大写字母、小写字母和数字。', passwordChangeHint: '验证当前密码后设置新密码；修改成功后，其他登录会话将被退出。', passwordChanged: '密码修改成功，其他登录会话已退出。' });
Object.assign(portalCopies['zh-TW'], { changePassword: '修改密碼', currentPassword: '目前密碼', passwordChangeHint: '驗證目前密碼後設定新密碼；修改成功後，其他登入工作階段將被登出。', passwordChanged: '密碼修改成功，其他登入工作階段已登出。' });
Object.assign(portalCopies['ja-JP'], { changePassword: 'パスワードを変更', currentPassword: '現在のパスワード', passwordChangeHint: '現在のパスワードを確認して新しいパスワードを設定します。他のセッションはログアウトされます。', passwordChanged: 'パスワードを変更し、他のセッションをログアウトしました。' });
Object.assign(portalCopies['ru-RU'], { changePassword: 'Изменить пароль', currentPassword: 'Текущий пароль', passwordChangeHint: 'Подтвердите текущий пароль и задайте новый. Остальные сеансы будут завершены.', passwordChanged: 'Пароль изменён. Остальные сеансы завершены.' });
Object.assign(portalCopies['vi-VN'], { changePassword: 'Đổi mật khẩu', currentPassword: 'Mật khẩu hiện tại', passwordChangeHint: 'Xác nhận mật khẩu hiện tại rồi đặt mật khẩu mới. Các phiên khác sẽ bị đăng xuất.', passwordChanged: 'Đã đổi mật khẩu và đăng xuất các phiên khác.' });
Object.assign(portalCopies['es-ES'], { changePassword: 'Cambiar contraseña', currentPassword: 'Contraseña actual', passwordChangeHint: 'Confirma la contraseña actual y elige una nueva. Se cerrarán las demás sesiones.', passwordChanged: 'Contraseña cambiada. Se cerraron las demás sesiones.' });
Object.assign(portalCopies['id-ID'], { changePassword: 'Ubah kata sandi', currentPassword: 'Kata sandi saat ini', passwordChangeHint: 'Konfirmasi kata sandi saat ini lalu pilih yang baru. Sesi lain akan dikeluarkan.', passwordChanged: 'Kata sandi diubah. Sesi lain telah dikeluarkan.' });
Object.assign(portalCopies['uk-UA'], { changePassword: 'Змінити пароль', currentPassword: 'Поточний пароль', passwordChangeHint: 'Підтвердьте поточний пароль і задайте новий. Інші сеанси буде завершено.', passwordChanged: 'Пароль змінено. Інші сеанси завершено.' });
Object.assign(portalCopies['tr-TR'], { changePassword: 'Parolayı değiştir', currentPassword: 'Mevcut parola', passwordChangeHint: 'Mevcut parolayı doğrulayın ve yeni bir parola belirleyin. Diğer oturumlar kapatılır.', passwordChanged: 'Parola değiştirildi. Diğer oturumlar kapatıldı.' });
Object.assign(portalCopies['pt-BR'], { changePassword: 'Alterar senha', currentPassword: 'Senha atual', passwordChangeHint: 'Confirme a senha atual e defina uma nova. As outras sessões serão encerradas.', passwordChanged: 'Senha alterada. As outras sessões foram encerradas.' });
Object.assign(portalCopies['ar-EG'], { changePassword: 'تغيير كلمة المرور', currentPassword: 'كلمة المرور الحالية', passwordChangeHint: 'أكّد كلمة المرور الحالية ثم اختر كلمة مرور جديدة. سيتم تسجيل خروج الجلسات الأخرى.', passwordChanged: 'تم تغيير كلمة المرور وتسجيل خروج الجلسات الأخرى.' });
Object.assign(portalCopies['fa-IR'], { changePassword: 'تغییر گذرواژه', currentPassword: 'گذرواژه فعلی', passwordChangeHint: 'گذرواژه فعلی را تأیید و گذرواژه جدید را تعیین کنید. نشست‌های دیگر خارج می‌شوند.', passwordChanged: 'گذرواژه تغییر کرد و نشست‌های دیگر خارج شدند.' });

Object.assign(portalCopies['zh-CN'], { emailOrAdmin: 'Gmail 邮箱或管理员账号', renew: '续费', upgrade: '升级套餐', renewalDescription: '从当前到期时间继续延长订阅有效期。', upgradeDescription: '立即切换到更高套餐，并保留当前剩余有效期。', accountCenter: '个人中心', accountOverview: '账户概览', invitationRewards: '邀请奖励', inviteFriends: '邀请好友', invitationDescription: '分享你的专属邀请链接。佣金按当前站点规则计算，结算后直接进入账户余额。', invitationCode: '邀请码', invitationLink: '专属邀请链接', copyInvitation: '复制邀请链接', invitedUsers: '已邀请用户', commissionRate: '佣金比例', pendingCommission: '待结算佣金', earnedCommission: '累计佣金', commissionBalanceHint: '账户余额不能提现，只能在购买、续费或升级本站套餐时抵扣。', firstPaymentRule: '佣金仅按被邀请用户首次成功付款计算。', inviteCodePermanent: '你的邀请码长期有效。', startHere: '从这里开始', emptySubscriptionDescription: '先选择套餐；支付成功后，订阅链接和完整导入步骤会自动显示在这里。', subscriptionReadyTitle: '从购买到连接', subscriptionReadyDescription: '选择套餐、完成付款、安装官方客户端，再导入你的专属订阅。', planHelpTitle: '按需要选择套餐', planHelpDescription: '比较流量、设备上限和计费周期；支付确认后系统会自动开通。', guideStartTitle: '三步完成连接', guideStartDescription: '安装官方客户端、导入订阅，再选择节点连接。', troubleshooting: '连接排查', troubleshootingDescription: '优先刷新订阅；仍无法使用时可提交工单或联系 Telegram 客服。', clientPickerTitle: '选择适合设备的客户端', clientPickerDescription: 'Windows、macOS、Linux 使用桌面客户端，Android 或 iOS 使用对应官方移动端。', afterInstallTitle: '安装完成后', afterInstallDescription: '进入“我的订阅”复制专属链接，并按当前平台的教程完成导入。', supportCenterTitle: '帮助与客服', supportCenterDescription: '账户或订阅问题建议提交工单；服务通知和一般咨询可通过 Telegram 联系。', telegramSupport: 'Telegram 客服', responseTimeTitle: '工单记录长期保留', responseTimeDescription: '客服回复会保留在对应工单中，之后可以继续同一段对话。', faqTitle: '提交前请准备', faqDescription: '请写明设备、客户端名称和完整报错；不要发送密码或完整订阅链接。', paymentSecureTitle: '支付宝收款', paymentSecureDescription: '生成二维码前会再次确认应付金额。', activationTitle: '自动开通', activationDescription: '支付确认后，系统自动为账户开通订阅。', invitationCopied: '邀请链接已复制' });
Object.assign(portalCopies['zh-TW'], { emailOrAdmin: 'Gmail 信箱或管理員帳號', renew: '續費', upgrade: '升級方案', renewalDescription: '從目前到期時間繼續延長訂閱有效期。', upgradeDescription: '立即切換至更高方案，並保留目前剩餘有效期。', accountCenter: '個人中心', accountOverview: '帳戶總覽', invitationRewards: '邀請獎勵', inviteFriends: '邀請好友', invitationDescription: '分享專屬邀請連結；佣金依目前規則計算，結算後直接進入帳戶餘額。', invitationCode: '邀請碼', invitationLink: '專屬邀請連結', copyInvitation: '複製邀請連結', invitedUsers: '已邀請使用者', commissionRate: '佣金比例', pendingCommission: '待結算佣金', earnedCommission: '累計佣金', commissionBalanceHint: '帳戶餘額不可提領，只能在購買、續費或升級本站方案時折抵。', firstPaymentRule: '佣金僅依受邀使用者首次成功付款計算。', inviteCodePermanent: '你的邀請碼長期有效。', startHere: '從這裡開始', emptySubscriptionDescription: '先選擇方案；付款成功後，訂閱連結與完整匯入步驟會自動顯示在此。', subscriptionReadyTitle: '從購買到連線', subscriptionReadyDescription: '選擇方案、完成付款、安裝官方客戶端，再匯入專屬訂閱。', planHelpTitle: '依需求選擇方案', planHelpDescription: '比較流量、裝置上限與計費週期；付款確認後系統自動開通。', guideStartTitle: '三步完成連線', guideStartDescription: '安裝官方客戶端、匯入訂閱，再選擇節點連線。', troubleshooting: '連線排查', troubleshootingDescription: '優先重新整理訂閱；仍無法使用時可提交工單或聯絡 Telegram 客服。', clientPickerTitle: '選擇適合裝置的客戶端', clientPickerDescription: 'Windows、macOS、Linux 使用桌面客戶端，Android 或 iOS 使用對應官方行動版。', afterInstallTitle: '安裝完成後', afterInstallDescription: '進入「我的訂閱」複製專屬連結，並依目前平台教學完成匯入。', supportCenterTitle: '協助與客服', supportCenterDescription: '帳戶或訂閱問題建議提交工單；服務通知與一般諮詢可透過 Telegram 聯絡。', telegramSupport: 'Telegram 客服', responseTimeTitle: '工單記錄會保留', responseTimeDescription: '客服回覆會保留於對應工單，之後可繼續同一段對話。', faqTitle: '提交前請準備', faqDescription: '請填寫裝置、客戶端名稱與完整錯誤；請勿傳送密碼或完整訂閱連結。', paymentSecureTitle: '支付寶付款', paymentSecureDescription: '建立 QR Code 前會再次確認應付金額。', activationTitle: '自動開通', activationDescription: '付款確認後，系統自動為帳戶開通訂閱。', invitationCopied: '邀請連結已複製' });
Object.assign(portalCopies['ja-JP'], { renew: '更新', upgrade: 'プランをアップグレード', renewalDescription: '現在の有効期限から契約期間を延長します。', upgradeDescription: '残りの有効期間を維持したまま上位プランへ切り替えます。' });
Object.assign(portalCopies['ru-RU'], { renew: 'Продлить', upgrade: 'Повысить тариф', renewalDescription: 'Продлить подписку от текущей даты окончания.', upgradeDescription: 'Сразу перейти на более высокий тариф, сохранив оставшийся срок.' });
Object.assign(portalCopies['vi-VN'], { renew: 'Gia hạn', upgrade: 'Nâng cấp gói', renewalDescription: 'Gia hạn từ ngày hết hạn hiện tại.', upgradeDescription: 'Chuyển ngay sang gói cao hơn và giữ thời gian còn lại.' });
Object.assign(portalCopies['es-ES'], { renew: 'Renovar', upgrade: 'Mejorar plan', renewalDescription: 'Extiende la suscripción desde su fecha de vencimiento actual.', upgradeDescription: 'Cambia ahora a un plan superior conservando el tiempo restante.' });
Object.assign(portalCopies['id-ID'], { renew: 'Perpanjang', upgrade: 'Tingkatkan paket', renewalDescription: 'Perpanjang langganan dari tanggal kedaluwarsa saat ini.', upgradeDescription: 'Beralih sekarang ke paket lebih tinggi sambil mempertahankan sisa waktu.' });
Object.assign(portalCopies['uk-UA'], { renew: 'Продовжити', upgrade: 'Підвищити тариф', renewalDescription: 'Продовжити підписку від поточної дати завершення.', upgradeDescription: 'Одразу перейти на вищий тариф зі збереженням решти строку.' });
Object.assign(portalCopies['tr-TR'], { renew: 'Yenile', upgrade: 'Paketi yükselt', renewalDescription: 'Aboneliği mevcut bitiş tarihinden itibaren uzatır.', upgradeDescription: 'Kalan süreyi koruyarak hemen daha yüksek pakete geçer.' });
Object.assign(portalCopies['pt-BR'], { renew: 'Renovar', upgrade: 'Melhorar plano', renewalDescription: 'Estende a assinatura a partir da data de vencimento atual.', upgradeDescription: 'Muda agora para um plano superior mantendo o período restante.' });
Object.assign(portalCopies['ar-EG'], { renew: 'تجديد', upgrade: 'ترقية الباقة', renewalDescription: 'تمديد الاشتراك ابتداءً من تاريخ الانتهاء الحالي.', upgradeDescription: 'الانتقال فورًا إلى باقة أعلى مع الاحتفاظ بالمدة المتبقية.' });
Object.assign(portalCopies['fa-IR'], { renew: 'تمدید', upgrade: 'ارتقای طرح', renewalDescription: 'اشتراک از تاریخ انقضای فعلی تمدید می‌شود.', upgradeDescription: 'با حفظ زمان باقی‌مانده فوراً به طرح بالاتر بروید.' });

// Keep the automatic-provisioning claim accurate in every supported locale.
portalCopies['zh-TW'].automaticActivation = '付款成功後自動開通';
portalCopies['ja-JP'].automaticActivation = '支払い後に自動開通';
portalCopies['ru-RU'].automaticActivation = 'Автоматическая активация после оплаты';
portalCopies['vi-VN'].automaticActivation = 'Tự động kích hoạt sau khi thanh toán';
portalCopies['es-ES'].automaticActivation = 'Activación automática después del pago';
portalCopies['id-ID'].automaticActivation = 'Aktivasi otomatis setelah pembayaran';
portalCopies['uk-UA'].automaticActivation = 'Автоматична активація після оплати';
portalCopies['tr-TR'].automaticActivation = 'Ödeme sonrası otomatik etkinleştirme';
portalCopies['pt-BR'].automaticActivation = 'Ativação automática após o pagamento';
portalCopies['ar-EG'].automaticActivation = 'تفعيل تلقائي بعد الدفع';
portalCopies['fa-IR'].automaticActivation = 'فعال‌سازی خودکار پس از پرداخت';

const neutralHeroBadges: Record<PortalLocale, string> = {
  'zh-CN': '安全稳定的连接服务',
  'zh-TW': '安全穩定的連線服務',
  'en-US': 'Secure and reliable access',
  'ja-JP': '安全で安定した接続サービス',
  'ru-RU': 'Безопасное и стабильное подключение',
  'vi-VN': 'Kết nối an toàn và ổn định',
  'es-ES': 'Acceso seguro y estable',
  'id-ID': 'Koneksi aman dan stabil',
  'uk-UA': 'Безпечне та стабільне підключення',
  'tr-TR': 'Güvenli ve kararlı bağlantı',
  'pt-BR': 'Acesso seguro e estável',
  'ar-EG': 'اتصال آمن ومستقر',
  'fa-IR': 'اتصال امن و پایدار',
};

for (const [locale, heroBadge] of Object.entries(neutralHeroBadges) as Array<[PortalLocale, string]>) {
  portalCopies[locale].heroBadge = heroBadge;
}

// Public-facing language should explain customer value and service expectations,
// never expose control-panel roles or infrastructure implementation details.
Object.assign(portalCopies['en-US'], {
  email: 'Email address',
  emailOrAdmin: 'Email address or account name',
  heroBadge: 'Reliable access, clear control',
  heroTitle: 'Stay connected across every device, without the complexity',
  heroDescription: 'Manage your plan, private subscription and device setup from one account. Usage, expiry and service updates stay easy to find.',
  serviceTitle: 'Everything you need, in one dependable place',
  serviceDescription: 'Plan details, device guides, subscription management and support are organized around your account.',
  platformGuides: 'supported platforms',
  subscriptionFormats: 'service commitments',
  selfService: 'account visibility',
  benefitsTitle: 'A dependable experience from checkout to daily use',
  benefitsDescription: 'Every important step comes with clear guidance, account visibility and practical security reminders.',
  fastTitle: 'Consistent service across your devices',
  fastDescription: 'Your plan stays ready across supported devices. Refresh the subscription whenever a configuration update is available.',
  privacyTitle: 'Your access stays under your control',
  privacyDescription: 'Your private link belongs to your account and can be replaced quickly if it is ever exposed.',
  simpleTitle: 'Clear guidance on every platform',
  simpleDescription: 'Step-by-step instructions are available for Windows, macOS, Android, iOS and Linux.',
  quickStart: 'Get started in three clear steps',
  chooseClient: 'Choose the app for your device',
  connect: 'Connect and start using the service',
  billingNotice: 'Plan price, data allowance and device limit are shown before checkout. Setup details appear in My subscription after payment is confirmed.',
  recommended: 'Most popular',
  automaticActivation: 'Available after payment confirmation',
  guidesDescription: 'Choose your device, then follow the matching installation and import instructions.',
  clientsDescription: 'Download links point to official release channels so you receive authentic updates and security fixes.',
  signInForTickets: 'Sign in to submit a question and follow its progress',
  gmailOnly: 'Register with an email verification code',
  choosePaymentMethodDescription: 'Choose any payment option currently available for this order.',
  resetDescription: 'Verify your email address, then set a new password. Your old password is not required.',
  noSubscription: 'You have not activated a subscription yet',
  emptySubscriptionDescription: 'After you choose a plan and complete payment, your private subscription, usage, expiry and setup steps will appear here.',
  subscriptionReadyTitle: 'Your setup, all in one place',
  subscriptionReadyDescription: 'Plan details, private subscription and device instructions stay together for easy access.',
  planHelpTitle: 'Choose the plan that fits how you use it',
  planHelpDescription: 'Compare data allowance, supported devices and billing periods before you pay. Every order remains visible in your account.',
  guideStartTitle: 'Start with your device, finish in three steps',
  guideStartDescription: 'Install a trusted app, import your private subscription, then connect when you are ready.',
  clientPickerTitle: 'Trusted apps for every supported device',
  clientPickerDescription: 'Choose the recommended desktop or mobile app for your operating system and follow the matching guide.',
  supportCenterTitle: 'Help when you need it',
  supportCenterDescription: 'Use a support ticket for account or subscription questions, and Telegram for general enquiries when available.',
  responseTimeTitle: 'Every question has a clear history',
  responseTimeDescription: 'Your messages, replies and progress stay together so you can return to the conversation at any time.',
  faqTitle: 'Share the right details for faster help',
  faqDescription: 'Include your device, app, time of the issue and exact error. Never include your password or full subscription link.',
  paymentSecureTitle: 'Clear, secure checkout',
  paymentSecureDescription: 'Your plan, billing period and final amount are confirmed before you continue to payment.',
  activationTitle: 'Activation progress you can follow',
  activationDescription: 'Once payment is confirmed, the subscription status and setup details are available from your account.',
});

Object.assign(portalCopies['zh-CN'], {
  email: '邮箱',
  emailOrAdmin: '邮箱或账号',
  heroBadge: '稳定、私密、清晰可控',
  heroTitle: '稳定连接你的每一台设备，日常使用更从容',
  heroDescription: '一个账户集中管理套餐、订阅与多端使用。流量、有效期、服务状态和使用指引随时可查。',
  serviceTitle: '从开通到日常使用，都有清晰指引',
  serviceDescription: '套餐权益、设备教程、订阅管理与问题支持集中在一个账户中，重要信息随时可查。',
  platformGuides: '个平台完整适配',
  subscriptionFormats: '项服务保障',
  selfService: '账户信息可查',
  benefitsTitle: '值得信赖的日常连接体验',
  benefitsDescription: '从支付、开通到多设备使用，每个关键环节都有明确说明与安全提示。',
  fastTitle: '稳定可用的多端体验',
  fastDescription: '套餐覆盖的服务会持续维护；配置更新时，刷新订阅即可同步到设备。',
  privacyTitle: '订阅信息由你掌控',
  privacyDescription: '专属链接仅绑定你的账户；如有泄露，可随时更换以保护使用安全。',
  simpleTitle: '每台设备都有清晰指引',
  simpleDescription: 'Windows、macOS、Android、iOS 与 Linux 均提供对应安装与导入步骤，减少试错。',
  quickStart: '三步开始使用',
  chooseClient: '选择适合设备的客户端',
  connect: '连接并开始使用',
  billingNotice: '套餐价格、流量与设备数量均清晰列明；付款确认后，可在“我的订阅”中查看开通状态与配置。',
  recommended: '多数用户选择',
  automaticActivation: '付款确认后开通',
  guidesDescription: '按设备查看对应的安装、导入与连接步骤，重要操作都有明确说明。',
  clientsDescription: '下载入口均指向官方发布渠道，便于获得完整更新与安全修复。',
  signInForTickets: '登录后即可提交问题，并随时查看处理进度',
  gmailOnly: '使用邮箱验证码注册',
  choosePaymentMethodDescription: '请选择当前订单支持的支付方式继续付款。',
  resetDescription: '验证邮箱后直接设置新密码，不需要提供原密码。',
  noSubscription: '你还未开通订阅',
  emptySubscriptionDescription: '选购并完成付款后，专属订阅、流量、有效期与导入步骤都会集中显示在这里。',
  subscriptionReadyTitle: '开通信息集中管理',
  subscriptionReadyDescription: '套餐权益、专属订阅与设备指引都会保留在账户中，需要时随时查看。',
  planHelpTitle: '按使用场景选择合适套餐',
  planHelpDescription: '付款前可比较流量、设备数量与计费周期；订单状态和开通进度会保留在账户中。',
  guideStartTitle: '从设备开始，三步完成设置',
  guideStartDescription: '安装可信客户端、导入专属订阅，再按指引完成连接。',
  clientPickerTitle: '为每种设备选择可信客户端',
  clientPickerDescription: '根据 Windows、macOS、Linux、Android 或 iOS 选择对应客户端，并查看匹配的使用步骤。',
  supportCenterTitle: '需要帮助时，我们始终有清晰入口',
  supportCenterDescription: '账户与订阅问题可提交工单；如已开放 Telegram，也可用于一般咨询与服务通知。',
  responseTimeTitle: '每个问题都有完整记录',
  responseTimeDescription: '提交内容、客服回复与后续进展会集中保留，方便随时继续沟通。',
  faqTitle: '提供这些信息，处理会更快',
  faqDescription: '请说明设备、客户端、出现时间与报错内容；请勿提交密码或完整订阅链接。',
  paymentSecureTitle: '支付信息清晰确认',
  paymentSecureDescription: '继续付款前会再次核对套餐、计费周期与最终应付金额。',
  activationTitle: '开通进度清晰可查',
  activationDescription: '付款确认后，可在订单与订阅页面查看处理状态和使用信息。',
});

Object.assign(portalCopies['zh-TW'], {
  email: '電子信箱',
  emailOrAdmin: '電子信箱或帳號',
  heroBadge: '穩定、私密、清楚可控',
  heroTitle: '穩定連接每一台裝置，日常使用更從容',
  heroDescription: '一個帳戶集中管理方案、訂閱與多裝置使用；流量、有效期限、服務狀態與使用說明隨時可查。',
  serviceTitle: '從開通到日常使用，都有清楚指引',
  serviceDescription: '方案權益、裝置教學、訂閱管理與問題支援集中在同一帳戶中。',
  platformGuides: '個平台完整支援',
  subscriptionFormats: '項服務保障',
  selfService: '帳戶資訊可查',
  benefitsTitle: '值得信賴的日常連線體驗',
  benefitsDescription: '從付款、開通到多裝置使用，每個重要環節都有明確說明與安全提醒。',
  fastTitle: '穩定可用的多裝置體驗',
  fastDescription: '方案涵蓋的服務會持續維護；設定更新時，重新整理訂閱即可同步至裝置。',
  privacyTitle: '訂閱資訊由你掌控',
  privacyDescription: '專屬連結只綁定你的帳戶；如有外洩，可隨時更換以保障使用安全。',
  simpleTitle: '每台裝置都有清楚指引',
  simpleDescription: 'Windows、macOS、Android、iOS 與 Linux 都提供對應安裝與匯入步驟。',
  billingNotice: '方案價格、流量與裝置數量均清楚列明；付款確認後可在「我的訂閱」查看開通狀態。',
  recommended: '多數使用者選擇',
  clientsDescription: '下載入口均指向官方發布管道，方便取得完整更新與安全修正。',
  choosePaymentMethodDescription: '請選擇目前訂單支援的付款方式繼續。',
  signInForTickets: '登入後即可提交問題，並隨時查看處理進度',
  noSubscription: '你尚未開通訂閱',
  planHelpTitle: '依使用情境選擇合適方案',
  planHelpDescription: '付款前可比較流量、裝置數量與計費週期；訂單與開通進度會保留在帳戶中。',
  paymentSecureTitle: '付款資訊清楚確認',
  paymentSecureDescription: '繼續付款前會再次核對方案、計費週期與最終應付金額。',
  activationTitle: '開通進度清楚可查',
  activationDescription: '付款確認後，可在訂單與訂閱頁面查看處理狀態與使用資訊。',
});

// Registration availability is controlled by the configured email-domain allowlist.
// Keep every public locale provider-neutral so adding a non-Gmail domain never leaves
// stale Gmail-only wording in login, registration, reset, or account screens.
const neutralEmailCopies: Record<PortalLocale, Pick<PortalCopy, 'email' | 'emailOrAdmin' | 'gmailOnly' | 'resetDescription'>> = {
  'en-US': { email: 'Email address', emailOrAdmin: 'Email address or account name', gmailOnly: 'Register with an email verification code', resetDescription: 'Verify your email address, then set a new password. Your old password is not required.' },
  'zh-CN': { email: '邮箱', emailOrAdmin: '邮箱或账号', gmailOnly: '使用邮箱验证码注册', resetDescription: '验证邮箱后直接设置新密码，不需要提供原密码。' },
  'zh-TW': { email: '電子信箱', emailOrAdmin: '電子信箱或帳號', gmailOnly: '使用電子信箱驗證碼註冊', resetDescription: '驗證電子信箱後直接設定新密碼，不需要提供原密碼。' },
  'ja-JP': { email: 'メールアドレス', emailOrAdmin: 'メールアドレスまたはアカウント名', gmailOnly: 'メール確認コードで登録', resetDescription: 'メールアドレスを確認して新しいパスワードを設定します。以前のパスワードは不要です。' },
  'ru-RU': { email: 'Адрес электронной почты', emailOrAdmin: 'Адрес электронной почты или имя аккаунта', gmailOnly: 'Регистрация по коду из письма', resetDescription: 'Подтвердите адрес электронной почты и задайте новый пароль. Старый пароль не требуется.' },
  'vi-VN': { email: 'Địa chỉ email', emailOrAdmin: 'Email hoặc tên tài khoản', gmailOnly: 'Đăng ký bằng mã xác minh email', resetDescription: 'Xác minh địa chỉ email rồi đặt mật khẩu mới. Không cần mật khẩu cũ.' },
  'es-ES': { email: 'Correo electrónico', emailOrAdmin: 'Correo electrónico o nombre de cuenta', gmailOnly: 'Regístrate con un código enviado por correo', resetDescription: 'Verifica tu correo y establece una contraseña nueva. No necesitas la contraseña anterior.' },
  'id-ID': { email: 'Alamat email', emailOrAdmin: 'Email atau nama akun', gmailOnly: 'Daftar dengan kode verifikasi email', resetDescription: 'Verifikasi alamat email lalu buat kata sandi baru. Kata sandi lama tidak diperlukan.' },
  'uk-UA': { email: 'Адреса електронної пошти', emailOrAdmin: 'Електронна пошта або ім’я акаунта', gmailOnly: 'Реєстрація за кодом з листа', resetDescription: 'Підтвердьте електронну пошту та задайте новий пароль. Старий пароль не потрібен.' },
  'tr-TR': { email: 'E-posta adresi', emailOrAdmin: 'E-posta veya hesap adı', gmailOnly: 'E-posta doğrulama koduyla kaydolun', resetDescription: 'E-posta adresinizi doğrulayın ve yeni parola belirleyin. Eski parolanız gerekmez.' },
  'pt-BR': { email: 'Endereço de e-mail', emailOrAdmin: 'E-mail ou nome da conta', gmailOnly: 'Cadastre-se com um código enviado por e-mail', resetDescription: 'Confirme seu e-mail e defina uma nova senha. A senha antiga não é necessária.' },
  'ar-EG': { email: 'البريد الإلكتروني', emailOrAdmin: 'البريد الإلكتروني أو اسم الحساب', gmailOnly: 'التسجيل باستخدام رمز تحقق عبر البريد', resetDescription: 'تحقق من بريدك الإلكتروني ثم عيّن كلمة مرور جديدة. لا تحتاج إلى كلمة المرور القديمة.' },
  'fa-IR': { email: 'نشانی ایمیل', emailOrAdmin: 'ایمیل یا نام حساب', gmailOnly: 'ثبت‌نام با کد تأیید ایمیل', resetDescription: 'ایمیل خود را تأیید و گذرواژه جدید تعیین کنید. نیازی به گذرواژه قبلی نیست.' },
};

for (const [locale, copy] of Object.entries(neutralEmailCopies) as Array<[PortalLocale, (typeof neutralEmailCopies)[PortalLocale]]>) {
  Object.assign(portalCopies[locale], copy);
}

const directDownloadCopies: Record<PortalLocale, Pick<PortalCopy, 'officialDownload' | 'guidesDescription' | 'clientsDescription' | 'chooseClient' | 'clientPickerDescription' | 'clientStepDescription' | 'simpleDescription'>> = {
  'en-US': { officialDownload: 'Download', guidesDescription: 'Download the app for your device, then follow the matching import guide.', clientsDescription: 'Installation packages are provided directly by this site.', chooseClient: 'Choose an app', clientPickerDescription: 'Download the installer that matches your device.', clientStepDescription: 'Download and install a supported app', simpleDescription: 'Direct downloads and step-by-step setup guides are available for every supported platform.' },
  'zh-CN': { officialDownload: '下载', guidesDescription: '按设备下载客户端，再查看对应的订阅导入教程。', clientsDescription: '安装包由本站直接提供，无需跳转到其他网站。', chooseClient: '选择客户端', clientPickerDescription: '选择与你的设备匹配的安装包并下载。', clientStepDescription: '下载并安装支持的客户端', simpleDescription: '各平台均提供站内下载与清晰的安装、导入步骤。' },
  'zh-TW': { officialDownload: '下載', guidesDescription: '依裝置下載客戶端，再查看對應的訂閱匯入教學。', clientsDescription: '安裝包由本站直接提供，無需前往其他網站。', chooseClient: '選擇客戶端', clientPickerDescription: '選擇與你的裝置相符的安裝包並下載。', clientStepDescription: '下載並安裝支援的客戶端', simpleDescription: '各平台均提供站內下載與清楚的安裝、匯入步驟。' },
  'ja-JP': { officialDownload: 'ダウンロード', guidesDescription: '端末に合うアプリをダウンロードし、対応するインポート手順を確認してください。', clientsDescription: 'インストーラーはこのサイトから直接ダウンロードできます。', chooseClient: 'アプリを選択', clientPickerDescription: '端末に合うインストーラーを選んでください。', clientStepDescription: '対応アプリをダウンロードしてインストール', simpleDescription: '対応する各プラットフォーム向けに直接ダウンロードと手順を用意しています。' },
  'ru-RU': { officialDownload: 'Скачать', guidesDescription: 'Скачайте приложение для своего устройства и следуйте инструкции по импорту.', clientsDescription: 'Установочные файлы доступны напрямую на этом сайте.', chooseClient: 'Выбрать приложение', clientPickerDescription: 'Скачайте установщик для своей платформы.', clientStepDescription: 'Скачайте и установите приложение', simpleDescription: 'Для каждой поддерживаемой платформы доступны загрузки и пошаговые инструкции.' },
  'vi-VN': { officialDownload: 'Tải xuống', guidesDescription: 'Tải ứng dụng phù hợp với thiết bị rồi làm theo hướng dẫn nhập.', clientsDescription: 'Gói cài đặt được tải trực tiếp từ trang này.', chooseClient: 'Chọn ứng dụng', clientPickerDescription: 'Tải gói cài đặt phù hợp với thiết bị của bạn.', clientStepDescription: 'Tải và cài đặt ứng dụng được hỗ trợ', simpleDescription: 'Mỗi nền tảng đều có tệp tải trực tiếp và hướng dẫn từng bước.' },
  'es-ES': { officialDownload: 'Descargar', guidesDescription: 'Descarga la aplicación para tu dispositivo y sigue la guía de importación.', clientsDescription: 'Los instaladores se descargan directamente desde este sitio.', chooseClient: 'Elegir aplicación', clientPickerDescription: 'Descarga el instalador compatible con tu dispositivo.', clientStepDescription: 'Descarga e instala una aplicación compatible', simpleDescription: 'Hay descargas directas y guías paso a paso para cada plataforma compatible.' },
  'id-ID': { officialDownload: 'Unduh', guidesDescription: 'Unduh aplikasi untuk perangkat Anda lalu ikuti panduan impor.', clientsDescription: 'Paket instalasi tersedia langsung dari situs ini.', chooseClient: 'Pilih aplikasi', clientPickerDescription: 'Unduh pemasang yang sesuai dengan perangkat Anda.', clientStepDescription: 'Unduh dan pasang aplikasi yang didukung', simpleDescription: 'Unduhan langsung dan panduan langkah demi langkah tersedia untuk setiap platform.' },
  'uk-UA': { officialDownload: 'Завантажити', guidesDescription: 'Завантажте застосунок для свого пристрою та виконайте інструкцію імпорту.', clientsDescription: 'Інсталяційні файли доступні безпосередньо на цьому сайті.', chooseClient: 'Вибрати застосунок', clientPickerDescription: 'Завантажте інсталятор для своєї платформи.', clientStepDescription: 'Завантажте та встановіть підтримуваний застосунок', simpleDescription: 'Для кожної платформи доступні прямі завантаження та покрокові інструкції.' },
  'tr-TR': { officialDownload: 'İndir', guidesDescription: 'Cihazınıza uygun uygulamayı indirin ve içe aktarma kılavuzunu izleyin.', clientsDescription: 'Kurulum paketleri doğrudan bu siteden sunulur.', chooseClient: 'Uygulama seç', clientPickerDescription: 'Cihazınıza uygun kurulum paketini indirin.', clientStepDescription: 'Desteklenen uygulamayı indirip kurun', simpleDescription: 'Her platform için doğrudan indirme ve adım adım kurulum kılavuzu bulunur.' },
  'pt-BR': { officialDownload: 'Baixar', guidesDescription: 'Baixe o aplicativo para seu dispositivo e siga o guia de importação.', clientsDescription: 'Os instaladores são fornecidos diretamente por este site.', chooseClient: 'Escolher aplicativo', clientPickerDescription: 'Baixe o instalador compatível com seu dispositivo.', clientStepDescription: 'Baixe e instale um aplicativo compatível', simpleDescription: 'Há downloads diretos e guias passo a passo para cada plataforma compatível.' },
  'ar-EG': { officialDownload: 'تنزيل', guidesDescription: 'نزّل التطبيق المناسب لجهازك ثم اتبع دليل الاستيراد.', clientsDescription: 'تتوفر حزم التثبيت مباشرة من هذا الموقع.', chooseClient: 'اختر التطبيق', clientPickerDescription: 'نزّل حزمة التثبيت المناسبة لجهازك.', clientStepDescription: 'نزّل وثبّت تطبيقًا مدعومًا', simpleDescription: 'تتوفر تنزيلات مباشرة وأدلة خطوة بخطوة لكل منصة مدعومة.' },
  'fa-IR': { officialDownload: 'دانلود', guidesDescription: 'برنامه مناسب دستگاه را دانلود کنید و راهنمای ورود را دنبال کنید.', clientsDescription: 'بسته‌های نصب مستقیماً از همین سایت ارائه می‌شوند.', chooseClient: 'انتخاب برنامه', clientPickerDescription: 'بسته نصب مناسب دستگاه خود را دانلود کنید.', clientStepDescription: 'یک برنامه پشتیبانی‌شده را دانلود و نصب کنید', simpleDescription: 'برای هر پلتفرم، دانلود مستقیم و راهنمای گام‌به‌گام در دسترس است.' },
};

for (const [locale, copy] of Object.entries(directDownloadCopies) as Array<[PortalLocale, (typeof directDownloadCopies)[PortalLocale]]>) {
  Object.assign(portalCopies[locale], copy);
}

const conversionCopies: Record<PortalLocale, Pick<PortalCopy,
  | 'heroBadge'
  | 'heroTitle'
  | 'heroDescription'
  | 'browsePlans'
  | 'readGuides'
  | 'serviceTitle'
  | 'serviceDescription'
  | 'platformGuides'
  | 'subscriptionFormats'
  | 'selfService'
  | 'benefitsTitle'
  | 'benefitsDescription'
  | 'fastTitle'
  | 'fastDescription'
  | 'privacyTitle'
  | 'privacyDescription'
  | 'simpleTitle'
  | 'simpleDescription'
  | 'billingNotice'
  | 'recommended'
  | 'traffic'
  | 'devices'
  | 'automaticActivation'
  | 'planHelpTitle'
  | 'planHelpDescription'
  | 'paymentSecureTitle'
  | 'paymentSecureDescription'
  | 'activationTitle'
  | 'activationDescription'
  | 'supportCenterTitle'
  | 'responseTimeTitle'
  | 'responseTimeDescription'
  | 'faqTitle'
  | 'faqDescription'
>> = {
  'en-US': { heroBadge: 'Built for cross-border work and global collaboration', heroTitle: 'Clear connection plans for efficient global work', heroDescription: 'Designed for cross-border commerce, overseas content operations and remote work, with plan-based data, device access and node permissions shown clearly before and after purchase.', browsePlans: 'Choose a plan now', readGuides: 'View setup guides', serviceTitle: 'Plans, devices and support in one place', serviceDescription: 'From purchase and activation to app installation and subscription upkeep, every important step stays easy to find.', platformGuides: 'platform guides', subscriptionFormats: 'subscription formats', selfService: 'self-service account', benefitsTitle: 'A connection experience built for global work', benefitsDescription: 'Each plan clearly sets out its data allowance, device limit and node access, so setup and ongoing use stay predictable.', fastTitle: 'Node access matched to your plan', fastDescription: 'Each plan uses its assigned node permissions. Refresh the subscription whenever available routes are updated.', privacyTitle: 'Keep your private subscription under control', privacyDescription: 'The link belongs to your account and can be rotated or revoked if it is exposed.', simpleTitle: 'Get started quickly on every platform', simpleDescription: 'On-site installers and step-by-step import guides are available for Windows, macOS, Android, iOS and Linux.', billingNotice: 'Price, billing period, data allowance and device limit are shown before purchase. Activation starts automatically after payment is confirmed.', recommended: 'Most popular', traffic: 'data per cycle', devices: 'devices', automaticActivation: 'Automatic activation after payment confirmation', planHelpTitle: 'Choose by workload, not guesswork', planHelpDescription: 'Compare price, billing period, data allowance and device limit. Node access follows the plan configuration shown after activation.', paymentSecureTitle: 'Clear purchase details', paymentSecureDescription: 'Review the plan, billing period and final amount before continuing to payment.', activationTitle: 'Automatic activation with visible progress', activationDescription: 'After payment is confirmed, activation begins automatically and its status remains visible in your account.', supportCenterTitle: 'Support with a clear history', responseTimeTitle: 'Keep every question in one place', responseTimeDescription: 'Ticket messages, replies and progress remain attached to your account for easy follow-up.', faqTitle: 'Prepare these details for faster help', faqDescription: 'Include your device, app, time of the issue and exact error. Never include your password or full subscription link.' },
  'zh-CN': { heroBadge: '为跨境业务与全球协作而设计', heroTitle: '跨境业务连接，套餐权益清晰可控', heroDescription: '面向海外内容运营、跨境电商与远程办公，按套餐提供对应流量、设备数量和节点权限，开通与使用信息清晰可查。', browsePlans: '立即选择套餐', readGuides: '查看使用文档', serviceTitle: '套餐、设备与服务，一站管理', serviceDescription: '从选购、开通到客户端安装与订阅维护，关键步骤集中呈现，日常管理更省心。', platformGuides: '个平台指引', subscriptionFormats: '种订阅格式', selfService: '自助账户管理', benefitsTitle: '为跨境工作打造的连接体验', benefitsDescription: '根据套餐获得对应流量、设备数和节点权限，信息透明，配置与维护更清晰。', fastTitle: '按套餐接入对应节点', fastDescription: '不同套餐对应各自节点权限，线路更新后刷新订阅即可同步到已连接设备。', privacyTitle: '专属订阅由你掌控', privacyDescription: '链接仅绑定你的账户；如有泄露，可在账户中轮换或吊销，降低继续暴露的风险。', simpleTitle: '多平台快速上手', simpleDescription: 'Windows、macOS、Android、iOS 与 Linux 均提供站内安装包和分步导入指引。', billingNotice: '套餐价格、计费周期、周期流量和设备上限均在购买前展示；付款确认后自动开通。', recommended: '最受欢迎', traffic: '周期流量', devices: '台设备上限', automaticActivation: '付款确认后自动开通', planHelpTitle: '按使用强度选择套餐', planHelpDescription: '对比价格、计费周期、周期流量和设备上限；节点权限以套餐开通后的账户信息为准。', paymentSecureTitle: '购买信息透明', paymentSecureDescription: '继续付款前，可再次核对套餐、计费周期与最终应付金额。', activationTitle: '付款确认后自动开通', activationDescription: '支付确认后系统自动开始开通，处理进度与结果均可在账户中查看。', supportCenterTitle: '问题处理全程可查', responseTimeTitle: '每个问题集中留档', responseTimeDescription: '工单内容、客服回复与处理进度都会保留在账户中，方便后续跟进。', faqTitle: '提交工单前请准备', faqDescription: '请注明设备、客户端、问题时间与报错内容；不要提供密码或完整订阅链接。' },
  'zh-TW': { heroBadge: '為跨境業務與全球協作而設計', heroTitle: '跨境業務連線，方案權益清楚可控', heroDescription: '面向海外內容營運、跨境電商與遠端工作，依方案提供對應流量、裝置數量與節點權限，開通與使用資訊清楚可查。', browsePlans: '立即選擇方案', readGuides: '查看使用文件', serviceTitle: '方案、裝置與服務，一站管理', serviceDescription: '從選購、開通到客戶端安裝與訂閱維護，重要步驟集中呈現，日常管理更省心。', platformGuides: '個平台指引', subscriptionFormats: '種訂閱格式', selfService: '自助帳戶管理', benefitsTitle: '為跨境工作打造的連線體驗', benefitsDescription: '依方案取得對應流量、裝置數與節點權限，資訊透明，設定與維護更清楚。', fastTitle: '依方案接入對應節點', fastDescription: '不同方案對應各自節點權限，線路更新後重新整理訂閱即可同步到裝置。', privacyTitle: '專屬訂閱由你掌控', privacyDescription: '連結只綁定你的帳戶；如有外洩，可在帳戶中輪換或撤銷。', simpleTitle: '多平台快速上手', simpleDescription: 'Windows、macOS、Android、iOS 與 Linux 均提供站內安裝包與分步匯入指引。', billingNotice: '方案價格、計費週期、週期流量與裝置上限均在購買前展示；付款確認後自動開通。', recommended: '最受歡迎', traffic: '週期流量', devices: '部裝置上限', automaticActivation: '付款確認後自動開通', planHelpTitle: '依使用強度選擇方案', planHelpDescription: '比較價格、計費週期、週期流量與裝置上限；節點權限以開通後的帳戶資訊為準。', paymentSecureTitle: '購買資訊透明', paymentSecureDescription: '繼續付款前，可再次核對方案、計費週期與最終應付金額。', activationTitle: '付款確認後自動開通', activationDescription: '付款確認後系統自動開始開通，處理進度與結果可在帳戶中查看。', supportCenterTitle: '問題處理全程可查', responseTimeTitle: '每個問題集中留檔', responseTimeDescription: '工單內容、客服回覆與處理進度都會保留在帳戶中，方便後續跟進。', faqTitle: '提交工單前請準備', faqDescription: '請註明裝置、客戶端、問題時間與錯誤內容；不要提供密碼或完整訂閱連結。' },
  'ja-JP': { heroBadge: '越境ビジネスとグローバル協業のために', heroTitle: '世界との接続を安定させ、協業をより効率的に', heroDescription: '越境EC、海外向けコンテンツ運用、リモートワーク向けに、プランごとの通信量、端末数、ノード権限を明確に提供します。', browsePlans: '今すぐプランを選ぶ', readGuides: '利用ガイドを見る', serviceTitle: 'プラン、端末、サポートを一か所で管理', serviceDescription: '購入と開通からアプリのインストール、サブスクリプション管理まで、必要な手順をまとめて確認できます。', platformGuides: '種類の端末ガイド', subscriptionFormats: '種類の形式', selfService: 'セルフサービス管理', benefitsTitle: 'グローバルワークのための接続体験', benefitsDescription: '通信量、端末上限、ノード権限をプランごとに明示し、設定と日常利用を分かりやすくします。', fastTitle: 'プランに応じたノードアクセス', fastDescription: '各プランに設定されたノード権限を利用し、ルート更新時はサブスクリプションを更新できます。', privacyTitle: '専用サブスクリプションを自分で管理', privacyDescription: '専用リンクはアカウントに紐付き、漏えい時にはローテーションまたは無効化できます。', simpleTitle: '各プラットフォームですぐに開始', simpleDescription: 'Windows、macOS、Android、iOS、Linux向けにサイト内ダウンロードと手順を用意しています。', billingNotice: '料金、請求期間、通信量、端末上限は購入前に表示され、支払い確認後に自動開通します。', recommended: '人気プラン', traffic: '期間内通信量', devices: '台', automaticActivation: '支払い確認後に自動開通', planHelpTitle: '利用量に合わせてプランを選択', planHelpDescription: '料金、請求期間、通信量、端末上限を比較できます。ノード権限は開通後のアカウント情報で確認できます。', paymentSecureTitle: '購入内容を明確に確認', paymentSecureDescription: '支払い前にプラン、請求期間、最終金額をもう一度確認できます。', activationTitle: '支払い確認後に自動開通', activationDescription: '支払い確認後に開通処理が始まり、進捗と結果をアカウントで確認できます。', supportCenterTitle: '問い合わせ履歴を一元管理', responseTimeTitle: 'すべての質問を記録', responseTimeDescription: 'チケット内容、返信、進捗がアカウントに保存され、後から確認できます。', faqTitle: 'お問い合わせ前にご用意ください', faqDescription: '端末、アプリ、発生時刻、エラー内容を記載し、パスワードや完全なリンクは送らないでください。' },
  'ru-RU': { heroBadge: 'Для трансграничного бизнеса и совместной работы', heroTitle: 'Стабильнее связь, эффективнее международная работа', heroDescription: 'Для трансграничной торговли, зарубежного контента и удалённой работы: объём трафика, число устройств и доступ к узлам указаны для каждого тарифа.', browsePlans: 'Выбрать тариф', readGuides: 'Открыть инструкции', serviceTitle: 'Тарифы, устройства и поддержка в одном месте', serviceDescription: 'Покупка, активация, установка приложения и управление подпиской собраны в понятном личном кабинете.', platformGuides: 'инструкций для платформ', subscriptionFormats: 'формата подписки', selfService: 'самообслуживание', benefitsTitle: 'Подключение для международной работы', benefitsDescription: 'Трафик, лимит устройств и доступ к узлам указаны заранее, поэтому настройка и использование остаются предсказуемыми.', fastTitle: 'Доступ к узлам по вашему тарифу', fastDescription: 'Каждый тариф использует назначенные ему узлы; после обновления маршрутов достаточно обновить подписку.', privacyTitle: 'Личная подписка под вашим контролем', privacyDescription: 'Ссылка привязана к аккаунту и при утечке может быть заменена или отозвана.', simpleTitle: 'Быстрый старт на любой платформе', simpleDescription: 'Для Windows, macOS, Android, iOS и Linux доступны загрузки с сайта и пошаговые инструкции.', billingNotice: 'Цена, период, трафик и лимит устройств видны до покупки. После подтверждения оплаты начинается автоматическая активация.', recommended: 'Самый популярный', traffic: 'трафика за период', devices: 'устройств', automaticActivation: 'Автоматическая активация после оплаты', planHelpTitle: 'Выберите тариф по интенсивности использования', planHelpDescription: 'Сравните цену, период, трафик и устройства. Доступ к узлам отображается в аккаунте после активации.', paymentSecureTitle: 'Прозрачные условия покупки', paymentSecureDescription: 'Перед оплатой ещё раз проверьте тариф, период и итоговую сумму.', activationTitle: 'Автоматическая активация после оплаты', activationDescription: 'После подтверждения оплаты процесс запускается автоматически, а его статус виден в аккаунте.', supportCenterTitle: 'Вся история поддержки сохранена', responseTimeTitle: 'Каждый вопрос в одном месте', responseTimeDescription: 'Сообщения, ответы и ход обработки сохраняются в вашем аккаунте.', faqTitle: 'Что указать для быстрого ответа', faqDescription: 'Укажите устройство, приложение, время и текст ошибки. Не отправляйте пароль или полную ссылку подписки.' },
  'vi-VN': { heroBadge: 'Dành cho kinh doanh xuyên biên giới và cộng tác toàn cầu', heroTitle: 'Kết nối ổn định hơn, cộng tác toàn cầu hiệu quả hơn', heroDescription: 'Phù hợp cho thương mại xuyên biên giới, vận hành nội dung quốc tế và làm việc từ xa, với lưu lượng, số thiết bị và quyền truy cập nút rõ ràng theo từng gói.', browsePlans: 'Chọn gói ngay', readGuides: 'Xem tài liệu sử dụng', serviceTitle: 'Quản lý gói, thiết bị và hỗ trợ tại một nơi', serviceDescription: 'Từ mua và kích hoạt đến cài ứng dụng và quản lý đăng ký, mọi bước quan trọng đều dễ tìm.', platformGuides: 'hướng dẫn nền tảng', subscriptionFormats: 'định dạng đăng ký', selfService: 'tự quản lý tài khoản', benefitsTitle: 'Trải nghiệm kết nối cho công việc toàn cầu', benefitsDescription: 'Mỗi gói nêu rõ lưu lượng, giới hạn thiết bị và quyền truy cập nút để việc thiết lập dễ dự đoán hơn.', fastTitle: 'Truy cập nút theo gói đã chọn', fastDescription: 'Mỗi gói dùng quyền truy cập nút tương ứng; chỉ cần làm mới đăng ký khi tuyến được cập nhật.', privacyTitle: 'Tự kiểm soát đăng ký riêng', privacyDescription: 'Liên kết gắn với tài khoản và có thể đổi hoặc thu hồi nếu bị lộ.', simpleTitle: 'Bắt đầu nhanh trên nhiều nền tảng', simpleDescription: 'Có bộ cài trên trang và hướng dẫn từng bước cho Windows, macOS, Android, iOS và Linux.', billingNotice: 'Giá, chu kỳ, lưu lượng và giới hạn thiết bị được hiển thị trước khi mua; hệ thống tự kích hoạt sau khi xác nhận thanh toán.', recommended: 'Phổ biến nhất', traffic: 'lưu lượng mỗi chu kỳ', devices: 'thiết bị', automaticActivation: 'Tự kích hoạt sau khi xác nhận thanh toán', planHelpTitle: 'Chọn gói theo mức độ sử dụng', planHelpDescription: 'So sánh giá, chu kỳ, lưu lượng và thiết bị; quyền truy cập nút được hiển thị trong tài khoản sau khi kích hoạt.', paymentSecureTitle: 'Thông tin mua hàng minh bạch', paymentSecureDescription: 'Kiểm tra lại gói, chu kỳ và tổng tiền trước khi thanh toán.', activationTitle: 'Tự kích hoạt sau khi thanh toán', activationDescription: 'Sau khi thanh toán được xác nhận, quá trình kích hoạt tự bắt đầu và trạng thái hiển thị trong tài khoản.', supportCenterTitle: 'Theo dõi hỗ trợ rõ ràng', responseTimeTitle: 'Mọi câu hỏi được lưu tập trung', responseTimeDescription: 'Nội dung phiếu, phản hồi và tiến độ đều được lưu trong tài khoản.', faqTitle: 'Chuẩn bị thông tin trước khi gửi phiếu', faqDescription: 'Ghi rõ thiết bị, ứng dụng, thời điểm và lỗi; không gửi mật khẩu hoặc liên kết đăng ký đầy đủ.' },
  'es-ES': { heroBadge: 'Diseñado para negocios transfronterizos y colaboración global', heroTitle: 'Conexiones más estables para colaborar mejor en todo el mundo', heroDescription: 'Pensado para comercio transfronterizo, operaciones de contenido internacional y trabajo remoto, con datos, dispositivos y acceso a nodos definidos por plan.', browsePlans: 'Elegir un plan ahora', readGuides: 'Ver documentación', serviceTitle: 'Planes, dispositivos y soporte en un solo lugar', serviceDescription: 'Desde la compra y activación hasta la instalación y gestión de la suscripción, cada paso importante queda claro.', platformGuides: 'guías de plataforma', subscriptionFormats: 'formatos de suscripción', selfService: 'cuenta autoservicio', benefitsTitle: 'Una experiencia de conexión para el trabajo global', benefitsDescription: 'Cada plan muestra sus datos, límite de dispositivos y acceso a nodos para que la configuración sea predecible.', fastTitle: 'Acceso a nodos según tu plan', fastDescription: 'Cada plan usa los permisos de nodos asignados; actualiza la suscripción cuando cambien las rutas disponibles.', privacyTitle: 'Tu suscripción privada bajo control', privacyDescription: 'El enlace pertenece a tu cuenta y puede rotarse o revocarse si queda expuesto.', simpleTitle: 'Empieza rápido en cualquier plataforma', simpleDescription: 'Hay instaladores y guías paso a paso para Windows, macOS, Android, iOS y Linux.', billingNotice: 'El precio, periodo, datos y límite de dispositivos se muestran antes de comprar. La activación comienza al confirmarse el pago.', recommended: 'Más popular', traffic: 'datos por periodo', devices: 'dispositivos', automaticActivation: 'Activación automática tras confirmar el pago', planHelpTitle: 'Elige según tu nivel de uso', planHelpDescription: 'Compara precio, periodo, datos y dispositivos. El acceso a nodos aparece en tu cuenta tras la activación.', paymentSecureTitle: 'Información de compra transparente', paymentSecureDescription: 'Revisa el plan, el periodo y el importe final antes de pagar.', activationTitle: 'Activación automática tras el pago', activationDescription: 'Al confirmarse el pago, el proceso se inicia y su estado queda visible en tu cuenta.', supportCenterTitle: 'Soporte con historial claro', responseTimeTitle: 'Cada consulta queda registrada', responseTimeDescription: 'Los mensajes, respuestas y avances se conservan en tu cuenta.', faqTitle: 'Prepara estos datos para recibir ayuda', faqDescription: 'Indica dispositivo, aplicación, hora y error exacto. No envíes tu contraseña ni el enlace completo.' },
  'id-ID': { heroBadge: 'Dibuat untuk bisnis lintas batas dan kolaborasi global', heroTitle: 'Koneksi lebih stabil untuk kolaborasi global yang efisien', heroDescription: 'Untuk perdagangan lintas batas, operasi konten internasional, dan kerja jarak jauh, dengan kuota, perangkat, dan akses node yang jelas di setiap paket.', browsePlans: 'Pilih paket sekarang', readGuides: 'Lihat panduan penggunaan', serviceTitle: 'Paket, perangkat, dan dukungan dalam satu tempat', serviceDescription: 'Dari pembelian dan aktivasi hingga pemasangan aplikasi dan pengelolaan langganan, semua langkah penting mudah ditemukan.', platformGuides: 'panduan platform', subscriptionFormats: 'format langganan', selfService: 'akun mandiri', benefitsTitle: 'Pengalaman koneksi untuk pekerjaan global', benefitsDescription: 'Setiap paket menjelaskan kuota, batas perangkat, dan akses node agar penyiapan serta penggunaan lebih terukur.', fastTitle: 'Akses node sesuai paket', fastDescription: 'Setiap paket memakai izin node yang ditetapkan; segarkan langganan saat rute tersedia diperbarui.', privacyTitle: 'Langganan pribadi dalam kendali Anda', privacyDescription: 'Tautan terikat ke akun dan dapat diganti atau dicabut jika terekspos.', simpleTitle: 'Mulai cepat di berbagai platform', simpleDescription: 'Pemasang dan panduan bertahap tersedia untuk Windows, macOS, Android, iOS, dan Linux.', billingNotice: 'Harga, periode, kuota, dan batas perangkat tampil sebelum pembelian; aktivasi dimulai otomatis setelah pembayaran dikonfirmasi.', recommended: 'Paling populer', traffic: 'kuota per periode', devices: 'perangkat', automaticActivation: 'Aktivasi otomatis setelah pembayaran dikonfirmasi', planHelpTitle: 'Pilih paket sesuai intensitas penggunaan', planHelpDescription: 'Bandingkan harga, periode, kuota, dan perangkat. Akses node tampil di akun setelah aktivasi.', paymentSecureTitle: 'Informasi pembelian yang transparan', paymentSecureDescription: 'Periksa kembali paket, periode, dan jumlah akhir sebelum membayar.', activationTitle: 'Aktivasi otomatis setelah pembayaran', activationDescription: 'Setelah pembayaran dikonfirmasi, proses dimulai otomatis dan statusnya terlihat di akun.', supportCenterTitle: 'Riwayat dukungan yang jelas', responseTimeTitle: 'Setiap pertanyaan tersimpan', responseTimeDescription: 'Isi tiket, balasan, dan progres tetap tersimpan di akun Anda.', faqTitle: 'Siapkan informasi ini sebelum mengirim tiket', faqDescription: 'Cantumkan perangkat, aplikasi, waktu, dan kesalahan; jangan kirim kata sandi atau tautan langganan lengkap.' },
  'uk-UA': { heroBadge: 'Для транскордонного бізнесу та глобальної співпраці', heroTitle: 'Стабільніше з’єднання для ефективнішої глобальної роботи', heroDescription: 'Для транскордонної торгівлі, міжнародного контенту та віддаленої роботи: трафік, кількість пристроїв і доступ до вузлів визначені для кожного плану.', browsePlans: 'Обрати план', readGuides: 'Переглянути інструкції', serviceTitle: 'Плани, пристрої та підтримка в одному місці', serviceDescription: 'Від купівлі й активації до встановлення застосунку та керування підпискою — усі важливі кроки легко знайти.', platformGuides: 'інструкцій для платформ', subscriptionFormats: 'формати підписки', selfService: 'самообслуговування', benefitsTitle: 'З’єднання для глобальної роботи', benefitsDescription: 'Кожен план чітко показує трафік, ліміт пристроїв і доступ до вузлів, щоб налаштування було передбачуваним.', fastTitle: 'Доступ до вузлів відповідно до плану', fastDescription: 'Кожен план використовує призначені вузли; після оновлення маршрутів достатньо оновити підписку.', privacyTitle: 'Особиста підписка під вашим контролем', privacyDescription: 'Посилання прив’язане до акаунта й може бути замінене або відкликане у разі витоку.', simpleTitle: 'Швидкий старт на кожній платформі', simpleDescription: 'Для Windows, macOS, Android, iOS та Linux доступні завантаження із сайту й покрокові інструкції.', billingNotice: 'Ціна, період, трафік і ліміт пристроїв показані до купівлі. Після підтвердження оплати починається автоматична активація.', recommended: 'Найпопулярніший', traffic: 'трафіку за період', devices: 'пристроїв', automaticActivation: 'Автоматична активація після оплати', planHelpTitle: 'Обирайте план за інтенсивністю використання', planHelpDescription: 'Порівняйте ціну, період, трафік і пристрої. Доступ до вузлів відображається після активації.', paymentSecureTitle: 'Прозорі умови купівлі', paymentSecureDescription: 'Перед оплатою перевірте план, період і остаточну суму.', activationTitle: 'Автоматична активація після оплати', activationDescription: 'Після підтвердження оплати процес запускається автоматично, а статус видно в акаунті.', supportCenterTitle: 'Зрозуміла історія підтримки', responseTimeTitle: 'Кожне звернення збережене', responseTimeDescription: 'Повідомлення, відповіді та прогрес залишаються у вашому акаунті.', faqTitle: 'Підготуйте дані для швидшої допомоги', faqDescription: 'Вкажіть пристрій, застосунок, час і точну помилку. Не надсилайте пароль або повне посилання.' },
  'tr-TR': { heroBadge: 'Sınır ötesi iş ve küresel iş birliği için', heroTitle: 'Daha verimli küresel çalışma için daha kararlı bağlantı', heroDescription: 'Sınır ötesi ticaret, uluslararası içerik operasyonları ve uzaktan çalışma için plan bazlı trafik, cihaz ve düğüm erişimi açıkça gösterilir.', browsePlans: 'Şimdi plan seç', readGuides: 'Kullanım belgelerini gör', serviceTitle: 'Planlar, cihazlar ve destek tek yerde', serviceDescription: 'Satın alma ve etkinleştirmeden uygulama kurulumuna ve abonelik yönetimine kadar önemli adımlar kolayca bulunur.', platformGuides: 'platform kılavuzu', subscriptionFormats: 'abonelik biçimi', selfService: 'kendi kendine hesap yönetimi', benefitsTitle: 'Küresel çalışma için bağlantı deneyimi', benefitsDescription: 'Her plan trafik, cihaz sınırı ve düğüm erişimini açıkça gösterir; kurulum ve kullanım öngörülebilir kalır.', fastTitle: 'Planınıza uygun düğüm erişimi', fastDescription: 'Her plan kendisine atanan düğümleri kullanır; rotalar güncellendiğinde aboneliği yenilemeniz yeterlidir.', privacyTitle: 'Özel aboneliğiniz sizin kontrolünüzde', privacyDescription: 'Bağlantı hesabınıza bağlıdır ve açığa çıkarsa yenilenebilir veya iptal edilebilir.', simpleTitle: 'Her platformda hızlı başlangıç', simpleDescription: 'Windows, macOS, Android, iOS ve Linux için site içi indirmeler ve adım adım kılavuzlar bulunur.', billingNotice: 'Fiyat, dönem, trafik ve cihaz sınırı satın almadan önce gösterilir; ödeme onaylanınca otomatik etkinleştirme başlar.', recommended: 'En popüler', traffic: 'dönemlik trafik', devices: 'cihaz', automaticActivation: 'Ödeme onayından sonra otomatik etkinleştirme', planHelpTitle: 'Kullanım yoğunluğunuza göre plan seçin', planHelpDescription: 'Fiyatı, dönemi, trafiği ve cihazları karşılaştırın. Düğüm erişimi etkinleştirmeden sonra hesabınızda görünür.', paymentSecureTitle: 'Şeffaf satın alma bilgileri', paymentSecureDescription: 'Ödemeden önce planı, dönemi ve nihai tutarı tekrar kontrol edin.', activationTitle: 'Ödeme sonrası otomatik etkinleştirme', activationDescription: 'Ödeme onaylandığında süreç otomatik başlar ve durum hesabınızda görünür.', supportCenterTitle: 'Açık destek geçmişi', responseTimeTitle: 'Her soru tek yerde kayıtlı', responseTimeDescription: 'Destek kayıtları, yanıtlar ve ilerleme hesabınızda saklanır.', faqTitle: 'Daha hızlı yardım için bunları hazırlayın', faqDescription: 'Cihazı, uygulamayı, zamanı ve hatayı belirtin; parolanızı veya tam abonelik bağlantısını göndermeyin.' },
  'pt-BR': { heroBadge: 'Feito para negócios internacionais e colaboração global', heroTitle: 'Conexões mais estáveis para um trabalho global mais eficiente', heroDescription: 'Para comércio internacional, operações de conteúdo no exterior e trabalho remoto, com dados, dispositivos e acesso a nós definidos por plano.', browsePlans: 'Escolher um plano agora', readGuides: 'Ver documentação', serviceTitle: 'Planos, dispositivos e suporte em um só lugar', serviceDescription: 'Da compra e ativação à instalação do aplicativo e gestão da assinatura, cada etapa importante fica fácil de encontrar.', platformGuides: 'guias de plataforma', subscriptionFormats: 'formatos de assinatura', selfService: 'conta de autoatendimento', benefitsTitle: 'Uma experiência de conexão para o trabalho global', benefitsDescription: 'Cada plano mostra dados, limite de dispositivos e acesso a nós, tornando a configuração mais previsível.', fastTitle: 'Acesso a nós conforme o plano', fastDescription: 'Cada plano usa as permissões de nós atribuídas; atualize a assinatura quando as rotas forem alteradas.', privacyTitle: 'Sua assinatura privada sob controle', privacyDescription: 'O link pertence à sua conta e pode ser trocado ou revogado se for exposto.', simpleTitle: 'Comece rápido em qualquer plataforma', simpleDescription: 'Há instaladores e guias passo a passo para Windows, macOS, Android, iOS e Linux.', billingNotice: 'Preço, período, dados e limite de dispositivos aparecem antes da compra; a ativação começa após a confirmação do pagamento.', recommended: 'Mais popular', traffic: 'dados por período', devices: 'dispositivos', automaticActivation: 'Ativação automática após confirmar o pagamento', planHelpTitle: 'Escolha pelo nível de uso', planHelpDescription: 'Compare preço, período, dados e dispositivos. O acesso a nós aparece na conta após a ativação.', paymentSecureTitle: 'Informações de compra transparentes', paymentSecureDescription: 'Confira plano, período e valor final antes de pagar.', activationTitle: 'Ativação automática após o pagamento', activationDescription: 'Após a confirmação, o processo começa automaticamente e o status fica visível na conta.', supportCenterTitle: 'Suporte com histórico claro', responseTimeTitle: 'Cada dúvida fica registrada', responseTimeDescription: 'Mensagens, respostas e progresso permanecem salvos na sua conta.', faqTitle: 'Prepare estes dados para receber ajuda', faqDescription: 'Informe dispositivo, aplicativo, horário e erro exato. Não envie sua senha nem o link completo.' },
  'ar-EG': { heroBadge: 'مصمم للأعمال العابرة للحدود والتعاون العالمي', heroTitle: 'اتصال أكثر استقرارًا لعمل عالمي أكثر كفاءة', heroDescription: 'للتجارة العابرة للحدود وإدارة المحتوى الدولي والعمل عن بُعد، مع توضيح البيانات وعدد الأجهزة وصلاحيات العقد لكل باقة.', browsePlans: 'اختر باقة الآن', readGuides: 'عرض دليل الاستخدام', serviceTitle: 'الباقات والأجهزة والدعم في مكان واحد', serviceDescription: 'من الشراء والتفعيل إلى تثبيت التطبيق وإدارة الاشتراك، ستجد كل خطوة مهمة بوضوح.', platformGuides: 'أدلة للمنصات', subscriptionFormats: 'صيغ اشتراك', selfService: 'إدارة ذاتية للحساب', benefitsTitle: 'تجربة اتصال للعمل العالمي', benefitsDescription: 'توضح كل باقة البيانات وحد الأجهزة وصلاحيات العقد لتبقى عملية الإعداد والاستخدام متوقعة.', fastTitle: 'وصول إلى العقد حسب الباقة', fastDescription: 'تستخدم كل باقة العقد المخصصة لها؛ حدّث الاشتراك عند تحديث المسارات المتاحة.', privacyTitle: 'اشتراكك الخاص تحت سيطرتك', privacyDescription: 'الرابط مرتبط بحسابك ويمكن تدويره أو إلغاؤه إذا انكشف.', simpleTitle: 'ابدأ بسرعة على كل منصة', simpleDescription: 'تتوفر ملفات تثبيت وأدلة خطوة بخطوة لويندوز وmacOS وأندرويد وiOS ولينكس.', billingNotice: 'يظهر السعر والدورة والبيانات وحد الأجهزة قبل الشراء، ويبدأ التفعيل تلقائيًا بعد تأكيد الدفع.', recommended: 'الأكثر اختيارًا', traffic: 'بيانات لكل دورة', devices: 'أجهزة', automaticActivation: 'تفعيل تلقائي بعد تأكيد الدفع', planHelpTitle: 'اختر الباقة حسب كثافة الاستخدام', planHelpDescription: 'قارن السعر والدورة والبيانات والأجهزة. تظهر صلاحيات العقد في الحساب بعد التفعيل.', paymentSecureTitle: 'معلومات شراء واضحة', paymentSecureDescription: 'راجع الباقة والدورة والمبلغ النهائي قبل الدفع.', activationTitle: 'تفعيل تلقائي بعد الدفع', activationDescription: 'بعد تأكيد الدفع تبدأ العملية تلقائيًا وتظهر حالتها في حسابك.', supportCenterTitle: 'سجل دعم واضح', responseTimeTitle: 'كل سؤال محفوظ في مكان واحد', responseTimeDescription: 'تبقى رسائل التذاكر والردود والتقدم محفوظة في حسابك.', faqTitle: 'جهّز هذه المعلومات لمساعدة أسرع', faqDescription: 'اذكر الجهاز والتطبيق والوقت والخطأ بدقة، ولا ترسل كلمة المرور أو رابط الاشتراك الكامل.' },
  'fa-IR': { heroBadge: 'برای کسب‌وکار فرامرزی و همکاری جهانی', heroTitle: 'اتصال پایدارتر برای همکاری جهانی کارآمدتر', heroDescription: 'برای تجارت فرامرزی، مدیریت محتوای بین‌المللی و دورکاری؛ حجم داده، تعداد دستگاه و دسترسی نود در هر طرح شفاف است.', browsePlans: 'اکنون طرح را انتخاب کنید', readGuides: 'مشاهده راهنمای استفاده', serviceTitle: 'طرح‌ها، دستگاه‌ها و پشتیبانی در یکجا', serviceDescription: 'از خرید و فعال‌سازی تا نصب برنامه و مدیریت اشتراک، همه مراحل مهم به‌سادگی در دسترس است.', platformGuides: 'راهنمای پلتفرم', subscriptionFormats: 'قالب اشتراک', selfService: 'مدیریت خودکار حساب', benefitsTitle: 'تجربه اتصال برای کار جهانی', benefitsDescription: 'هر طرح حجم داده، محدودیت دستگاه و دسترسی نود را شفاف نشان می‌دهد تا راه‌اندازی قابل پیش‌بینی باشد.', fastTitle: 'دسترسی نود متناسب با طرح', fastDescription: 'هر طرح از نودهای تعیین‌شده استفاده می‌کند؛ پس از به‌روزرسانی مسیرها، اشتراک را تازه‌سازی کنید.', privacyTitle: 'اشتراک خصوصی زیر کنترل شما', privacyDescription: 'پیوند به حساب شما متصل است و در صورت افشا می‌توان آن را چرخاند یا لغو کرد.', simpleTitle: 'شروع سریع در هر پلتفرم', simpleDescription: 'فایل نصب و راهنمای مرحله‌ای برای Windows، macOS، Android، iOS و Linux موجود است.', billingNotice: 'قیمت، دوره، حجم داده و محدودیت دستگاه پیش از خرید نمایش داده می‌شود و فعال‌سازی پس از تأیید پرداخت خودکار آغاز می‌شود.', recommended: 'محبوب‌ترین', traffic: 'داده در هر دوره', devices: 'دستگاه', automaticActivation: 'فعال‌سازی خودکار پس از تأیید پرداخت', planHelpTitle: 'طرح را بر اساس میزان استفاده انتخاب کنید', planHelpDescription: 'قیمت، دوره، داده و دستگاه‌ها را مقایسه کنید. دسترسی نود پس از فعال‌سازی در حساب نمایش داده می‌شود.', paymentSecureTitle: 'اطلاعات خرید شفاف', paymentSecureDescription: 'پیش از پرداخت، طرح، دوره و مبلغ نهایی را دوباره بررسی کنید.', activationTitle: 'فعال‌سازی خودکار پس از پرداخت', activationDescription: 'پس از تأیید پرداخت، فرایند خودکار آغاز می‌شود و وضعیت آن در حساب قابل مشاهده است.', supportCenterTitle: 'سابقه روشن پشتیبانی', responseTimeTitle: 'هر پرسش در یکجا ثبت می‌شود', responseTimeDescription: 'پیام‌های تیکت، پاسخ‌ها و روند رسیدگی در حساب شما باقی می‌ماند.', faqTitle: 'برای دریافت کمک سریع‌تر آماده کنید', faqDescription: 'دستگاه، برنامه، زمان و خطای دقیق را بنویسید؛ گذرواژه یا پیوند کامل اشتراک را ارسال نکنید.' },
};

for (const [locale, copy] of Object.entries(conversionCopies) as Array<[PortalLocale, (typeof conversionCopies)[PortalLocale]]>) {
  Object.assign(portalCopies[locale], copy);
}

const conservativeHeroTitles: Record<PortalLocale, string> = {
  'en-US': 'Clear connection plans for efficient global work',
  'zh-CN': '跨境业务连接，套餐权益清晰可控',
  'zh-TW': '跨境業務連線，方案權益清楚可控',
  'ja-JP': '海外業務向けの接続を、分かりやすく管理',
  'ru-RU': 'Понятные тарифы для международной работы',
  'vi-VN': 'Gói kết nối rõ ràng cho công việc toàn cầu',
  'es-ES': 'Planes de conexión claros para el trabajo global',
  'id-ID': 'Paket koneksi yang jelas untuk pekerjaan global',
  'uk-UA': 'Зрозумілі плани підключення для глобальної роботи',
  'tr-TR': 'Küresel çalışma için açık bağlantı planları',
  'pt-BR': 'Planos de conexão claros para o trabalho global',
  'ar-EG': 'باقات اتصال واضحة للعمل العالمي',
  'fa-IR': 'طرح‌های اتصال شفاف برای کار جهانی',
};

for (const [locale, title] of Object.entries(conservativeHeroTitles) as Array<[PortalLocale, string]>) {
  portalCopies[locale].heroTitle = title;
}

portalCopies['zh-CN'].relatedPlan = '关联套餐';
portalCopies['zh-CN'].relatedPlanHint = '请选择出现问题的套餐，便于客服快速核对订阅配置。';
portalCopies['zh-TW'].relatedPlan = '關聯方案';
portalCopies['zh-TW'].relatedPlanHint = '請選擇出現問題的方案，方便客服快速核對訂閱設定。';

export interface PlanComparisonCopy {
  title: string;
  description: string;
  benefit: string;
  scenario: string;
  monthlyTraffic: string;
  peakSpeed: string;
  globalCoverage: string;
  standardNodes: string;
  advancedNodes: string;
  premiumRoutes: string;
  residentialRelay: string;
  residentialIpSale: string;
  socialMedia: string;
  crossBorderWork: string;
  liveStreaming: string;
  uploadOptimization: string;
  peakPriority: string;
  failover: string;
  devices: string;
  support: string;
  included: string;
  notIncluded: string;
  notConfigured: string;
  upTo: (count: number) => string;
  deviceCount: (count: number) => string;
  tiers: Array<{
    name: string;
    scenario: string;
    advancedNodes: string;
    premiumRoutes: string;
    socialMedia: string;
    crossBorderWork: string;
    liveStreaming: string;
    uploadOptimization: string;
    peakPriority: string;
    failover: string;
    support: string;
  }>;
}

const enPlanComparison: PlanComparisonCopy = {
  title: 'Plan benefits',
  description: 'Compare traffic, speed, route access and service levels before choosing.',
  benefit: 'Plan benefit',
  scenario: 'Best for',
  monthlyTraffic: 'Monthly traffic',
  peakSpeed: 'Peak speed',
  globalCoverage: '100+ countries and regions',
  standardNodes: 'Global standard nodes',
  advancedNodes: 'Advanced dedicated nodes',
  premiumRoutes: 'IEPL/IPLC high-speed routes',
  residentialRelay: 'Connect your own residential IP',
  residentialIpSale: 'Residential IP provided or sold',
  socialMedia: 'Overseas social media',
  crossBorderWork: 'Cross-border commerce and remote work',
  liveStreaming: 'Overseas live streaming',
  uploadOptimization: 'Live upload optimization',
  peakPriority: 'Peak-hour route priority',
  failover: 'Route failover',
  devices: 'Simultaneous devices',
  support: 'Support response',
  included: 'Included',
  notIncluded: 'Not included',
  notConfigured: 'Not configured',
  upTo: (count) => `Up to ${count}`,
  deviceCount: (count) => `${count} devices`,
  tiers: [
    { name: 'Global Access', scenario: 'Everyday personal use', advancedNodes: 'Not included', premiumRoutes: 'Not included', socialMedia: 'Everyday use', crossBorderWork: 'Basic use', liveStreaming: 'Not recommended', uploadOptimization: 'Not included', peakPriority: 'Standard', failover: 'Basic', support: 'Standard' },
    { name: 'Cross-border Growth', scenario: 'Cross-border business operations', advancedNodes: 'Partially available', premiumRoutes: 'Selected routes', socialMedia: 'Frequent operations', crossBorderWork: 'Recommended', liveStreaming: '1080p streaming', uploadOptimization: 'Included', peakPriority: 'Priority', failover: 'Fast switching', support: 'Priority' },
    { name: 'Streaming Flagship', scenario: 'Professional streaming and teams', advancedNodes: 'Full access', premiumRoutes: 'All routes', socialMedia: 'Professional operations', crossBorderWork: 'Professional', liveStreaming: '4K and long-duration streaming', uploadOptimization: 'High priority', peakPriority: 'Highest', failover: 'Priority switching', support: 'Dedicated priority' },
  ],
};

export const planComparisonCopies = Object.fromEntries(
  (Object.keys(portalCopies) as PortalLocale[]).map((locale) => [locale, enPlanComparison]),
) as Record<PortalLocale, PlanComparisonCopy>;

planComparisonCopies['zh-CN'] = {
  title: '套餐权益对比', description: '选购前对比流量、速率、线路权限与服务等级。', benefit: '套餐权益', scenario: '适用场景', monthlyTraffic: '每月流量', peakSpeed: '峰值速率', globalCoverage: '覆盖100+国家和地区', standardNodes: '全球标准节点', advancedNodes: '高级专线节点', premiumRoutes: 'IEPL/IPLC高速线路', residentialRelay: '接入自有住宅IP', residentialIpSale: '提供或销售住宅IP', socialMedia: '海外社交媒体', crossBorderWork: '跨境电商与远程办公', liveStreaming: '海外直播', uploadOptimization: '直播上行优化', peakPriority: '高峰期线路优先级', failover: '线路故障切换', devices: '同时在线设备', support: '客服响应', included: '包含', notIncluded: '不包含', notConfigured: '未设置', upTo: (count) => `最多${count}个`, deviceCount: (count) => `${count}台`,
  tiers: [
    { name: '全球畅游版', scenario: '日常个人使用', advancedNodes: '—', premiumRoutes: '—', socialMedia: '日常使用', crossBorderWork: '基础使用', liveStreaming: '不推荐', uploadOptimization: '—', peakPriority: '标准', failover: '基础', support: '标准' },
    { name: '跨境增长版', scenario: '跨境业务运营', advancedNodes: '部分开放', premiumRoutes: '精选线路', socialMedia: '高频运营', crossBorderWork: '推荐', liveStreaming: '1080P直播', uploadOptimization: '包含', peakPriority: '优先', failover: '快速切换', support: '优先' },
    { name: '直播旗舰版', scenario: '专业直播及团队', advancedNodes: '全部开放', premiumRoutes: '全部线路', socialMedia: '专业运营', crossBorderWork: '专业级', liveStreaming: '4K及长时间直播', uploadOptimization: '高优先级', peakPriority: '最高', failover: '优先切换', support: '专属优先' },
  ],
};

planComparisonCopies['zh-TW'] = {
  ...planComparisonCopies['zh-CN'], title: '方案權益比較', description: '選購前比較流量、速率、線路權限與服務等級。', benefit: '方案權益', scenario: '適用場景', monthlyTraffic: '每月流量', peakSpeed: '峰值速率', globalCoverage: '覆蓋100+國家和地區', standardNodes: '全球標準節點', advancedNodes: '高級專線節點', premiumRoutes: 'IEPL/IPLC高速線路', residentialRelay: '接入自有住宅IP', residentialIpSale: '提供或銷售住宅IP', socialMedia: '海外社群媒體', crossBorderWork: '跨境電商與遠端辦公', liveStreaming: '海外直播', uploadOptimization: '直播上行優化', peakPriority: '高峰期線路優先級', failover: '線路故障切換', devices: '同時在線裝置', support: '客服回應', included: '包含', notIncluded: '不包含', notConfigured: '未設定', upTo: (count) => `最多${count}個`, deviceCount: (count) => `${count}台`,
  tiers: [
    { name: '全球暢遊版', scenario: '日常個人使用', advancedNodes: '—', premiumRoutes: '—', socialMedia: '日常使用', crossBorderWork: '基礎使用', liveStreaming: '不建議', uploadOptimization: '—', peakPriority: '標準', failover: '基礎', support: '標準' },
    { name: '跨境增長版', scenario: '跨境業務營運', advancedNodes: '部分開放', premiumRoutes: '精選線路', socialMedia: '高頻營運', crossBorderWork: '推薦', liveStreaming: '1080P直播', uploadOptimization: '包含', peakPriority: '優先', failover: '快速切換', support: '優先' },
    { name: '直播旗艦版', scenario: '專業直播及團隊', advancedNodes: '全部開放', premiumRoutes: '全部線路', socialMedia: '專業營運', crossBorderWork: '專業級', liveStreaming: '4K及長時間直播', uploadOptimization: '高優先級', peakPriority: '最高', failover: '優先切換', support: '專屬優先' },
  ],
};

export const localeOptions: Array<{ value: PortalLocale; label: string; shortLabel: string }> = [
  { value: 'zh-CN', label: 'CN · 简体中文', shortLabel: 'CN' },
  { value: 'zh-TW', label: 'TW · 繁體中文', shortLabel: 'TW' },
  { value: 'en-US', label: 'EN · English', shortLabel: 'EN' },
  { value: 'ja-JP', label: 'JP · 日本語', shortLabel: 'JP' },
  { value: 'ru-RU', label: 'RU · Русский', shortLabel: 'RU' },
  { value: 'vi-VN', label: 'VN · Tiếng Việt', shortLabel: 'VN' },
  { value: 'es-ES', label: 'ES · Español', shortLabel: 'ES' },
  { value: 'id-ID', label: 'ID · Bahasa Indonesia', shortLabel: 'ID' },
  { value: 'uk-UA', label: 'UA · Українська', shortLabel: 'UA' },
  { value: 'tr-TR', label: 'TR · Türkçe', shortLabel: 'TR' },
  { value: 'pt-BR', label: 'BR · Português', shortLabel: 'BR' },
  { value: 'ar-EG', label: 'AR · العربية', shortLabel: 'AR' },
  { value: 'fa-IR', label: 'FA · فارسی', shortLabel: 'FA' },
];
