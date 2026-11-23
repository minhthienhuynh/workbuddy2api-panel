package panel

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEnJSSyntax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping JS syntax check")
	}
	path, err := filepath.Abs("en.js")
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(node, "--check", path).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js syntax error:\n%s", out)
	}
}

func TestEnJSTranslationEngine(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node engine test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelectorAll: () => [],
    addEventListener: () => {}
  },
  console: console
};
sandbox.window = sandbox;
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const t = sandbox.__WB_EN__;
if (!t) throw new Error("__WB_EN__ not exposed in test context");

// Test static dictionary translations
const pairs = [
  ['控制台', 'Dashboard'],
  ['账号池', 'Account Pool'],
  ['任务中心', 'Task Center'],
  ['实时任务', 'Live Tasks'],
  ['模型与档位', 'Models & Effort'],
  ['用量分析', 'Usage Analytics'],
  ['配置', 'Settings'],
  ['运行日志', 'System Logs'],
  ['添加账号', 'Add Account'],
  ['一键测活', 'Check Health'],
  ['一键签到', 'Check-in All'],
  ['一键保活', 'Keepalive All'],
  ['保存配置', 'Save Settings'],
  ['冷却中', 'Cooling Down'],
  ['正常', 'Healthy']
];

for (const [zh, en] of pairs) {
  const got = t.translateText(zh);
  if (got !== en) {
    throw new Error('translateText("' + zh + '") = "' + got + '", want "' + en + '"');
  }
}

// Test dynamic regex rules
if (t.translateText('剩余 500 积分') !== 'Remaining 500 Credits') {
  throw new Error('Dynamic regex rule failed for credits: got ' + t.translateText('剩余 500 积分'));
}
if (t.translateText('10 秒前') !== '10 seconds ago') {
  throw new Error('Dynamic regex rule failed for relative time: got ' + t.translateText('10 秒前'));
}
console.log("OK");
`
	tmp, err := os.CreateTemp("", "en_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js engine test failed:\n%s", out)
	}
}

func TestEnJSRuntimeHooks(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node hooks test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');

let toastMsg = '';
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelectorAll: () => [],
    addEventListener: () => {}
  },
  console: console
};
sandbox.window = sandbox;
sandbox.window.toast = (msg) => { toastMsg = msg; };

vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const t = sandbox.__WB_EN__;
if (!t || typeof t.init !== 'function') {
  throw new Error("__WB_EN__.init is not a function");
}

t.init();

// 1. Title and Lang
if (sandbox.document.title !== 'WorkBuddy2API · Console') {
  throw new Error('title not updated, got: ' + sandbox.document.title);
}
if (sandbox.document.documentElement.lang !== 'en') {
  throw new Error('lang not updated, got: ' + sandbox.document.documentElement.lang);
}

// 2. Toast Hook
sandbox.window.toast('操作成功');
if (toastMsg !== 'Success') {
  throw new Error('toast message not translated, got: ' + toastMsg);
}

// 3. User input preservation
const input = {
  tagName: 'INPUT',
  placeholder: '请输入账号备注',
  value: '这是我的自定义备注',
  attributes: {},
  getAttribute: () => null,
  setAttribute: () => {}
};
t.translateNode(input);
if (input.placeholder !== 'Please enter account note' && !input.placeholder.includes('note')) {
  // Check if placeholder translated
  if (input.placeholder === '请输入账号备注') {
    throw new Error('placeholder not translated');
  }
}
if (input.value !== '这是我的自定义备注') {
  throw new Error('input.value was corrupted!');
}

console.log("HOOKS_OK");
`
	tmp, err := os.CreateTemp("", "en_hooks_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js runtime hooks test failed:\n%s", out)
	}
}

func TestEnJSReviewerFixes(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');

let confirmMsg = '';
let alertMsg = '';
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelector: (sel) => {
      if (sel === 'title') return { childNodes: [] };
      return null;
    },
    querySelectorAll: () => [],
    addEventListener: () => {}
  },
  console: console
};
sandbox.window = sandbox;
sandbox.window.confirm = (msg) => { confirmMsg = msg; return true; };
sandbox.window.alert = (msg) => { alertMsg = msg; };

vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const t = sandbox.__WB_EN__;
if (!t) throw new Error("__WB_EN__ not exposed");
t.init();

// 1. Confirm and Alert Dialog hooks
sandbox.window.confirm('确定要删除吗？');
if (confirmMsg === '确定要删除吗？') {
  throw new Error('confirm message not translated');
}
sandbox.window.alert('操作成功');
if (alertMsg !== 'Success') {
  throw new Error('alert message not translated, got: ' + alertMsg);
}

// 2. Node value mutation guard against infinite loop
let setCounter = 0;
const textNode = {
  nodeType: 3,
  _val: '未知文本abc',
  get nodeValue() { return this._val; },
  set nodeValue(v) { setCounter++; this._val = v; }
};
// If translateText returns same string or unchanged, nodeValue setter must not be invoked
t.translateNode(textNode);
if (setCounter > 0 && textNode._val === '未知文本abc') {
  throw new Error('nodeValue was assigned even though text did not change! setCounter=' + setCounter);
}

// 3. Script/Style tag exclusion
let scriptNodeTouched = false;
const scriptEl = {
  tagName: 'SCRIPT',
  nodeType: 1,
  childNodes: [{
    nodeType: 3,
    get nodeValue() { return 'console.log("提示");'; },
    set nodeValue(v) { scriptNodeTouched = true; }
  }]
};
scriptEl.firstChild = scriptEl.childNodes[0];
scriptEl.childNodes[0].nextSibling = null;
t.translateTree(scriptEl);
if (scriptNodeTouched) {
  throw new Error('translateTree touched script tag content!');
}

console.log("REVIEW_FIXES_OK");
`
	tmp, err := os.CreateTemp("", "en_review_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js reviewer fixes test failed:\n%s", out)
	}
}

func TestEnJSCheckinNotifications(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping check-in notifications test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelector: () => null,
    querySelectorAll: () => [],
    addEventListener: () => {}
  },
  console: console
};
sandbox.window = sandbox;
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const t = sandbox.__WB_EN__;
if (!t) throw new Error("__WB_EN__ not exposed");

const testCases = [
  "国际区账号不适用国内区签到",
  "该账号不适用国内区签到",
  "该账号不适用签到",
  "API Key 账号无签到体系（已跳过签到，仅刷新额度）",
  "企业账号无签到体系（已跳过签到，仅刷新额度）",
  "今天已签到，无需重复，上游不会再发积分",
  "今天已签到，无需重复，上游不会再发积分，当前积分 100/200",
  "今天已签到，无需重复，上游不会再发积分，当前积分 100",
  "签到未完成：上游还在处理，稍后重试",
  "签到未完成：上游还在处理，稍后重试，当前积分 100/200",
  "签到未完成：请稍后重试",
  "签到未完成：今天已签到，请明天再来",
  "签到未完成：网络连接超时",
  "签到未完成：活动未开启，当前积分 100/200",
  "签到未完成：企业账号不支持该操作",
  "签到未完成：请求处理中，请勿重复操作",
  "签到成功",
  "签到成功，+100 积分",
  "签到成功，+100 积分，当前积分 100/200",
  "签到成功，当前积分 100/200",
  "签到成功，+100 积分；余额刷新失败：401 Unauthorized",
  "签到成功；余额刷新失败：401 Unauthorized",
  "组内签到：新签到 0 个，不适用 1 个",
  "组内签到：新签到 1 个，今日已签 2 个，不适用 1 个",
  "组内签到：新签到 0 个，不适用 1 个，失败 1 个（fa351ba0：国际区账号不适用国内区签到）",
  "当前分组里没有需要「签到」的账号",
  "全部签到已开始，结果见日志",
  "已有全量任务在执行中，请等待本轮完成后再触发",
  "对分组「VIP」的 5 个账号执行「签到」？",
  "已签",
  "今日已签到：重复签发不了积分，点击只是重查状态并刷新余额",
  "退出选号，但照常签到 / 活跃上报 / 保活 / 刷新余额",
  "退出选号，但照常保活 / 刷新额度",
  "给这个账号起个本机备注（只存浏览器）",
  "只读探测上游是否已解封：不 revive、不参与选号、不改任何状态",
  "刷新企业版已分配额度（上游 get-enterprise-user-usage）",
  "签到失败",
  "保活失败",
  "余额查询失败",
  "猫猫旅行失败",
  "活跃上报失败",
  "账号资源查询失败",
  "连登兑换/抽奖失败",
  "抽奖失败",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版活动状态查询失败（不发签到请求）: connection refused",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版签到活动未开启（每日 100 积分），跳过",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版今天已签到（连登 5 天，每日 100 积分）",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版今天已签到（幂等）",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版签到失败: 400 Bad Request",
  "2026/11/23 15:04:05 checkin fa351ba0: 国际版签到成功（每日 100 积分）",
  "2026/11/23 15:04:05 WARN: checkin u1(fa351ba0): 活动状态查询失败（照常签到）: timeout",
  "2026/11/23 15:04:05 WARN: checkin u1(fa351ba0): 上游签到活动当前不生效（截止 2026-12-31），仍尝试签到",
  "2026/11/23 15:04:05 checkin u1(fa351ba0): 上游显示今天已签到（连登 10 天，活动至 2026-12-31）",
  "2026/11/23 15:04:05 checkin u1(fa351ba0): 今天已签到（幂等）",
  "2026/11/23 15:04:05 checkin u1(fa351ba0): 签到成功"
];

const reChinese = /[\u4e00-\u9fa5]/;
for (const s of testCases) {
  const tr = t.translateText(s);
  if (reChinese.test(tr)) {
    throw new Error('Untranslated Chinese in: "' + s + '" => "' + tr + '"');
  }
}
console.log("CHECKIN_NOTIFICATIONS_OK");
`
	tmp, err := os.CreateTemp("", "en_checkin_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js check-in notifications test failed:\n%s", out)
	}
}

func TestEnJSTaskModalAndCenter(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelector: () => null,
    querySelectorAll: () => [],
    addEventListener: () => {}
  },
  console: console
};
sandbox.window = sandbox;
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const trEngine = sandbox.__WB_EN__;
if (!trEngine) throw new Error("__WB_EN__ not exposed");

const testCases = [
  // 1. Task Modal header, notes, and states
  "账号任务",
  "操作随账号能力开放；「领取」仅在进度达标后可用。",
  "国际区任务暂仅支持查询，请在官方客户端操作",
  "国际区任务当前仅支持查询，请在官方客户端完成。",
  "国际区任务仅支持查询，不执行国内区成长任务",
  "该账号无个人成长体系（企业版或静态 API Key），不支持成长任务",
  "该账号无个人成长体系（企业版或静态 API Key）；请改用个人账号",
  "无个人成长体系（企业版或静态 API Key）",
  "该账号暂无任务",
  "查看本次执行结果",
  "等待后台任务开始处理",
  "后台进度查询失败：connection timeout",

  // 2. Table header and body elements
  "任务",
  "进度",
  "奖励",
  "状态",
  "未提供",
  "已领取",
  "可领取",
  "未解锁",
  "未接受",
  "可在客户端完成",
  "进行中",
  "已完成",
  "领取",
  "一键完成",
  "接受",
  "待执行",
  "等待计分",
  "领奖待重试",
  "执行失败",
  "排队",
  "跳过",

  // 3. Table rewards and jump tooltips
  "+100 分",
  "+5 能",
  "+100 分 +5 能",
  "+300 分 +8 能 Buddy",
  "+100 分 Buddy",
  "跳转：https://workbuddy.cn",
  "完成一次对话\n使用 WorkBuddy 完成至少一次对话交互\n跳转：https://workbuddy.cn",

  // 4. All 25 AUTO_TASKS values
  "上报 5 条对话活跃事件（自动补足差额）",
  "上报解锁 → 同意协议 → 领取第一只 Buddy",
  "桌面指纹事件链上报（已验证可点亮）",
  "上报「进入 Buddy 应用」事件链（已验证可点亮）",
  "上报「定时任务创建」事件（已验证可点亮）",
  "上报「读资料库介绍」事件（已验证可点亮）",
  "上报「使用模板创建任务」事件组 ×5（三账号实测点亮）",
  "上报「灵感案例做同款发送 Prompt」事件组（三账号实测点亮）",
  "真实专家召唤+使用链 ×5（专家市场+真实 chat，三账号实测点亮）",
  "真实专家团召唤+使用链 ×3（三账号实测点亮）",
  "设置主题 API + 皮肤生效事件（两账号实测点亮）",
  "真实轻量云专家召唤+使用链（真实对话 requestId，两账号实测点亮）",
  "真实「无畏契约拼豆专家」召唤+使用链（真实对话 requestId，截止 11-15）",
  "「中国电商运营专家」召唤+使用链（专家页搜索口径，截止 11-20，+50 分）",
  "「抖音策略师」召唤+使用链（专家页搜索口径，截止 11-20，+50 分）",
  "「商家宣传页与海报设计师」召唤+使用链（专家页搜索口径，截止 11-20，+50 分）",
  "「直播电商教练」召唤+使用链（专家页搜索口径，截止 11-20，+50 分）",
  "真实对话 + skill_info 技能加载事件（实测点亮）",
  "小程序首对话（小程序口径）：accept → mini 对话上报 → 领奖（+100c+5e）",
  "小程序选专家对话（小程序口径）：市场专家 id → accept → expert_actual_use 上报 → 领奖（+200c+5e）",
  "小程序五次对话（小程序口径）：accept → mini 对话上报 ×5（自动补差额）→ 领奖（+300c+5e）",
  "小程序定时任务（预留，每日零点解锁一环）：accept → 定时任务创建事件 → 领奖（判据待解锁验证）",
  "小程序使用 GLM5.2（预留）：accept → 带模型字段的 mini 对话上报 → 领奖（判据待解锁验证）",
  "小程序十次对话（预留）：accept → mini 对话上报 ×target（自动补差额）→ 领奖",
  "体验灵感功能（预留，疑 PC 口径）：accept → 灵感事件组（PC+mp 双形态）→ 领奖（判据待解锁验证）",

  // 5. Buttons and dialog actions
  "全部接受",
  "后台完成可自动任务",
  "重新查询",
  "关闭",
  "接受中…",
  "启动中…",
  "执行中…",
  "后台执行中…",

  // 6. Job presentation compound strings
  "排队中 · 处理 0 / 5 项 · 等待后台执行 · 已完成 0 项",
  "执行中 · 处理 2 / 5 项 · 正在处理：chat_5 · 已完成 1 项",
  "已结束 · 处理 5 / 5 项 · 全部可自动任务已处理 · 已完成 5 项",
  "已中断 · 处理 1 / 5 项 · 服务关闭，任务已中断 · 已完成 1 项",
  "执行失败 · 处理 1 / 5 项 · 后台任务异常: context canceled · 失败 1 项",
  "已完成 1 项，报名已提交 2 批，等待计分 3 项，领奖待重试 1 项，跳过 2 项，失败 1 项",

  // 7. Task actions and outcomes
  "已领取奖励（+100c +5e）",
  "已领取奖励（+100c）",
  "任务点亮并领取奖励（+300c +8e）",
  "已补报 5 条对话事件",
  "已设置主题并上报皮肤生效事件",
  "已上报 template_used ×5",
  "已上报 playbook_cta_click + playbook_prompt_send",
  "已上报 buddyapp 进入五连事件（同时覆盖 Buddy_App 与 Buddy_App_QQ）",
  "已上报定时任务创建事件",
  "已上报资料库介绍阅读事件",
  "已按桌面端指纹上报完整对话事件链（agent_task_created→chat_response）",
  "已上报真实对话 + skill_info 技能加载事件",
  "；已自动领奖 +100 分 +5 能",
  "；领奖接口已确认（奖励此前可能已领取）",
  "；服务端已确认领取",
  "；服务端进度尚未达标，等待入账后再确认",
  "；回读时未找到该任务，暂不能确认完成",
  "；任务已达标，但领奖失败: upstream 500 error",
  "；回读进度失败: connection timeout",
  "执行失败: rate limited",
  "查询任务失败: 502 Bad Gateway",
  "后台任务已启动，关闭页面后仍会继续执行",
  "该账号已有后台任务，已恢复进度显示",
  "已接受 5 个任务",
  "已接受 2 个，1 个被上游拒绝（可重试）",
  "所有任务均已接受"
];

const reChinese = /[\u4e00-\u9fa5]/;
for (const s of testCases) {
  const tr = trEngine.translateText(s);
  if (reChinese.test(tr)) {
    throw new Error('Untranslated Chinese in: "' + s + '" => "' + tr + '"');
  }
}
console.log("TASK_MODAL_AND_CENTER_OK");
`
	tmp, err := os.CreateTemp("", "en_task_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js task modal and center test failed:\n%s", out)
	}
}

func TestEnJSSupplementaryReviewFindings(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelector: () => null,
    querySelectorAll: () => []
  }
};
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const trEngine = sandbox.window.__WB_EN__;
if (!trEngine) throw new Error('window.__WB_EN__ is not defined');

const testCases = [
  // Account pool buttons
  "解冻",
  "探活",
  "恢复选号",
  "暂停选号",
  "重试",

  // Status badges & tags
  "熔断 · 1时05分",
  "连败降权 · 10秒",
  "积分冷却 · 2分30秒",
  "限流冷却 · 45秒",
  "已暂停选号",
  "保留积分",
  "疑似登录失效 · 待重登",

  // Countdowns & durations
  "预计 2分30秒 后重试",
  "预计 1时05分 后重试",
  "预计 45秒 后重试",
  "预计 2026-11-23 18:00 解封（剩余 10分00秒）",
  "等待重新探测",
  "上游重置：2026-11-23 18:00",
  "网关最早重试：2026-11-23 18:00",

  // Disabled kinds & reasons
  "上游封禁",
  "需在官方申诉，revive 无效",
  "登录失效",
  "需重新登录",
  "鉴权失败",
  "确认后重登或恢复",
  "手动禁用",
  "随时可恢复",
  "刚刚",

  // Empty states
  "账号池是空的",
  "点击右上角「添加账号」，用浏览器登录一个 WorkBuddy 账号",
  "暂无国内版账号",
  "当前账号池内均为国际版账号（共 5 个）",
  "暂无国际版账号",
  "当前账号池内均为国内版账号（共 3 个）",

  // Enterprise & pricing
  "不限",
  "企业版不限量（上游 limitNum=-1）",
  "实测单价",
  "免费",
  "积分（相对池内最高）",
  "已禁用：可用 0，冻结快照 5000",

  // Placeholder & collision fixes
  "昵称（可选）",
  "uid（可选，缺省由 key 派生）",
  "（可选）",
  "可选 / 总数",

  // Proxies view
  "5 个代理 · 3 个可用",
  "添加 2 个 · 重复 1 个 · 无效 0 个",
  "第 3 行：格式错误",
  "代理池为空",
  "出口 1.2.3.4 · JP 住宅",
  "出口 5.6.7.8 · US 机房",
  "默认随机",
  "直连",
  "已设置密码",
  "检测中",
  "先解除账号绑定",

  // Balance trends
  "暂无余额采样。服务每 30 分钟自动记录一点，余额刷新后即出现（也可点上方「刷新」立即采样一次）。",
  "暂无按日数据（需跨过零点后才出现第一天）。",
  "该域暂无采样",
  "切到其他域，或等待下一次余额刷新",
  "读取余额趋势失败：网络连接超时",
  "已失效 2 个（不计入上方统计，仅保留历史记录）",
  "较上一点 +500",
  "⚠️ 该日采样不完整（当天未结束或中途启用）",
  "该区间无消耗记录（余额未下降过）",
  "当前窗口不足以覆盖整月，切到「近 30 天」才可看本月值",
  "其他",

  // Usage analytics & vouchers
  "暂无用量数据。发起一次对话后再刷新。",
  "读取用量失败：500 Internal Error",
  "读取用量失败",
  "已过期",
  "可使用",
  "有效期至 2026-12-31",
  "长期有效",
  "2026-11-20 抽中",
  "券码",
  "券",
  "二维码",
  "其余 5 个账号，展开",
  "收起其余 5 个账号",

  // Accessibility aria-label & buttons
  "打开导航",
  "切换主题",
  "按分组筛选",
  "编辑代理",
  "上一页",
  "下一页",
  "按域筛选",
  "按能力筛选",
  "按思考档位筛选",
  "按价格筛选",
  "排序方式",
  "搜索请求记录",
  "读取条数",
  "导出区间",
  "启用 proxy1",
  "检测 proxy1",
  "编辑 proxy1",
  "删除 proxy1",
  "先解除账号绑定 proxy1",
  "最近一次：5 次 / 12.3k / 延迟 120ms / 45.2 tok/s",
  "1 accounts · 5 Todos · 0 可领 · just updated",
  "Scanned  412 行 · 4 类 · 截至 01:38:27",
  "08:33:59 growth 后台Task：后台Task queued",
  "08:33:54 scheduler: BalanceBackgroundRefresh每 5m0s（Pause中Display 0s）",
  "13:24:29 panel: 手动全量Keepalive已触发",
  "13:24:30 panel: pause uid=fa351ba0-5d7b-4f0b-b883-2c53a8d36693（Pause Selection，保号Task照常）",
  "13:24:31 panel: resume uid=fa351ba0-5d7b-4f0b-b883-2c53a8d36693（Resume参与Select Account）",
  "13:27:45 panel: Settings已Save（热生效完成；Restart Required字段 0 个）",
  "5 条 · 净 −9"
];

const reChinese = /[\u4e00-\u9fa5]/;
for (const s of testCases) {
  const tr = trEngine.translateText(s);
  if (reChinese.test(tr)) {
    throw new Error('Untranslated Chinese in: "' + s + '" => "' + tr + '"');
  }
}

// Explicit assertions on key translations
if (trEngine.translateText("昵称（可选）") !== "Nickname (optional)") {
  throw new Error("Expected 'Nickname (optional)', got: " + trEngine.translateText("昵称（可选）"));
}
if (trEngine.translateText("可选 / 总数") !== "Available / Total") {
  throw new Error("Expected 'Available / Total', got: " + trEngine.translateText("可选 / 总数"));
}
if (trEngine.translateText("5 个代理 · 3 个可用") !== "5 proxies · 3 available") {
  throw new Error("Expected '5 proxies · 3 available', got: " + trEngine.translateText("5 个代理 · 3 个可用"));
}
if (trEngine.translateText("解冻") !== "Revive") {
  throw new Error("Expected 'Revive', got: " + trEngine.translateText("解冻"));
}
if (trEngine.translateText("探活") !== "Probe Ban") {
  throw new Error("Expected 'Probe Ban', got: " + trEngine.translateText("探活"));
}
if (trEngine.translateText("已暂停选号") !== "Paused Selection") {
  throw new Error("Expected 'Paused Selection', got: " + trEngine.translateText("已暂停选号"));
}

console.log("SUPPLEMENTARY_REVIEW_FINDINGS_OK");
`
	tmp, err := os.CreateTemp("", "en_supp_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js supplementary review findings test failed:\n%s", out)
	}
}

func TestEnJSRequestAndSystemLogs(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping Node test")
	}
	script := `
const fs = require('fs');
const vm = require('vm');

const src = fs.readFileSync('en.js', 'utf8');
const sandbox = {
  window: {},
  document: {
    documentElement: { lang: 'zh-CN' },
    title: 'WorkBuddy2API',
    body: { childNodes: [] },
    querySelector: () => null,
    querySelectorAll: () => []
  }
};
vm.createContext(sandbox);
vm.runInContext(src, sandbox);

const trEngine = sandbox.window.__WB_EN__;
if (!trEngine) throw new Error('window.__WB_EN__ is not defined');

const testCases = [
  // Request Logs (#view-events)
  "请求事件日志",
  "共 0 条事件",
  "共 42 条事件",
  "筛选、查看并分析实时调用事件与性能指标（完全对齐 Keeper 风格）。",
  "清空归档",
  "删掉归档目录里的全部请求事件文件（不可恢复）",
  "导出 JSON",
  "模型",
  "全部",
  "全部 (5)",
  "来源",
  "状态",
  "成功",
  "429 限流",
  "错误",
  "清除筛选",
  "展示",
  "最近 20 条",
  "最近 1000 条",
  "刷新",
  "时间",
  "推理强度",
  "结果",
  "请求",
  "延时",
  "生成速度",
  "Token",
  "缓存",
  "执行器",
  "客户端 IP",
  "用户代理",
  "加载中...",
  "读取失败",
  "请求归档未开启，事件日志不可用",
  "读取失败: network error",
  "没有符合筛选条件的请求事件",
  "502 错误",
  "首字延迟 1.2s",
  "思考 Token",
  "命中 1.50K",
  "未命中 200",
  "点击查看请求上下文",
  "客户端未携带思考档位，这是网关补的默认档",
  "客户端要求关闭思考，或该档位模型不支持",
  "左：客户端请求；右：网关实际发送",
  "本次请求没有记录上下文（在「配置 → 日志」打开「记录请求上下文」后新产生的请求才有）",
  "请求上下文",
  "清空归档会删掉归档目录里的全部请求事件文件（不可恢复，导出过的 JSON 不受影响）。确认清空？",
  "已清空归档：删除 3 个文件（1.2 MB）",
  "清空失败：permission denied",

  // System Logs (#view-logs) - Request History, Error Summary, System Logs, Credits History
  "请求记录",
  "调用来源：IP 取 X-Forwarded-For 首段，无代理头时回落 TCP 对端",
  "搜索 IP / UA / 模型 / 账号 / 请求 ID",
  "搜索请求记录",
  "按结果筛选",
  "全部结果",
  "HTTP 错误",
  "流错误",
  "中断",
  "读取条数",
  "重新读取",
  "来源 IP",
  "耗时",
  "积分",
  "请求 ID",
  "已完成",
  "完成成功率",
  "HTTP 成功率",
  "平均耗时",
  "进行中",
  "归档文件",
  "JSONL 归档 12.3 KB",
  "JSONL 归档 12.3 KB · 丢弃 2 条",
  "JSONL 归档 12.3 KB · 错误：disk full",
  "仅内存指标，JSONL 归档已关闭",
  "没有符合当前筛选条件的请求记录",
  "暂无请求记录",
  "命中 5 / 100 条",
  "命中 5 / 100 条 · 来源未记录",
  "100 条",
  "100 条 · 来源未记录",
  "200 成功",
  "500 HTTP 错误",
  "500 流错误",
  "499 中断",
  "12:00:00 | 200 成功 | gpt-5 | acct1 | 1.2.3.4 | curl/8.0 | 120ms | 1.5k tok | 0.12 credit | 命中 85.0% | req_123",
  "错误汇总",
  "按账号 × 类型聚合，最该看的排最前",
  "把当前表里的行记成已读（之后新出现的仍然正常显示）",
  "标为已读",
  "隐藏已读：关",
  "隐藏已读：开",
  "没有未读错误（3 条已读被隐藏）",
  "窗口内没有 WARN/ERR",
  "扫描 500 行 · 3 类 · 已读 1 · 隐藏已读 · 截至 12:00:00",
  "本机存储不可用，标为已读未生效",
  "客户端中断",
  "会话失效",
  "WAF 拦截",
  "上游 5xx",
  "账号授权/配额故障",
  "模型不可用",
  "内容拦截",
  "上游 404",
  "请求参数错误",
  "上游 custom_kind",
  "传输失败",
  "响应体读取失败",
  "上游超时（未罚号）",
  "WAF IP 级拦截",
  "凭证落盘失败",
  "上游空流",
  "首帧否决（换号）",
  "上游流解析失败",
  "续写降级",
  "续写救援（预告句）",
  "续写救援（说一句就停）",
  "续写救援（思考吃光预算）",
  "模型目录拉取异常",
  "积分保底让位",
  "缓存重算（费用）",
  "正文截断（finish=length）",
  "上游断流（已补截断终态）",
  "轮转退避中止",
  "模型限流(6004)",
  "模型不存在(11102)",
  "其他（未归类）",
  "运行日志",
  "每频道最近 500 行；已按天落盘，重启后自动回填，可按区间导出",
  "导出区间",
  "今天",
  "最近 3 天",
  "最近 7 天",
  "导出 ZIP",
  "自动滚动：开",
  "自动滚动：关",
  "暂无日志",
  "任务 10 · 对话 20 · 系统 5",
  "任务 10 行",
  "对话 20 行",
  "系统 5 行",
  "导出失败：日志落盘未启用（检查 logging.panel_log_enabled 与数据目录权限）",
  "导出失败：该区间没有落盘日志（可能是保留期外，或落盘刚启用）",
  "积分历史",
  "每次真实查到余额与上次比对，变动即留痕",
  "余额 1,234 → 1,200",
  "暂无积分变动记录",
  "积分历史不可用",
  "10 条 · 净 −1,234"
];

const reChinese = /[\u4e00-\u9fa5]/;
for (const s of testCases) {
  const tr = trEngine.translateText(s);
  if (reChinese.test(tr)) {
    throw new Error('Untranslated Chinese in: "' + s + '" => "' + tr + '"');
  }
}

const exact = [
  ["全部", "All"],
  ["全部 (5)", "All (5)"],
  ["读取失败", "Load failed"],
  ["首字延迟 1.2s", "TTFB 1.2s"],
  ["命中 1.50K", "Hit 1.50K"],
  ["未命中 200", "Miss 200"],
  ["命中 5 / 100 条 · 来源未记录", "Matched 5 / 100 · Source not recorded"],
  ["100 条 · 来源未记录", "100 items · Source not recorded"],
  ["12:00:00 | 200 成功 | gpt-5 | acct1 | 1.2.3.4 | curl/8.0 | 120ms | 1.5k tok | 0.12 credit | 命中 85.0% | req_123",
   "12:00:00 | 200 Success | gpt-5 | acct1 | 1.2.3.4 | curl/8.0 | 120ms | 1.5k tok | 0.12 credit | Hit 85.0% | req_123"],
  ["10 条 · 净 −1,234", "10 items · Net −1,234"]
];
for (const [zh, want] of exact) {
  const got = trEngine.translateText(zh);
  if (got !== want) {
    throw new Error('translateText("' + zh + '") = "' + got + '", want "' + want + '"');
  }
}

console.log("REQUEST_AND_SYSTEM_LOGS_OK");
`
	tmp, err := os.CreateTemp("", "en_logs_test_*.js")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	out, err := exec.Command(node, tmp.Name()).CombinedOutput()
	if err != nil {
		t.Fatalf("en.js request and system logs test failed:\n%s", out)
	}
}
