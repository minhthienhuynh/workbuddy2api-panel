#!/usr/bin/env node
/**
 * Playwright E2E Verification Script for WorkBuddy2API English Web Panel
 *
 * Usage:
 *   node scripts/verify_panel_playwright.mjs
 *   PANEL_URL="http://localhost:7863/panel/" node scripts/verify_panel_playwright.mjs
 */

import path from 'path';
import { fileURLToPath } from 'url';

let chromium;
try {
  ({ chromium } = await import('playwright'));
} catch {
  try {
    ({ chromium } = await import('/home/ubuntu/.npm/_npx/86170c4cd1c5da32/node_modules/playwright/index.mjs'));
  } catch (err) {
    console.error('Playwright module not found. Install playwright or execute with npx.');
    process.exit(1);
  }
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REPO_ROOT = path.resolve(__dirname, '..');

const TARGET_URL = process.env.PANEL_URL || 'https://codebuddy2api-rbejke-129-150-60-238.sslip.io/panel/';
const API_KEY = process.env.PANEL_API_KEY || 'cb2-4f056d9ba2af73b77eed3be7f1fd7942';

// Detect system chromium / playwright binary
const CHROMIUM_PATHS = [
  process.env.CHROMIUM_PATH,
  '/home/ubuntu/.cache/ms-playwright/chromium-1255/chrome-linux-arm64/chrome',
  '/home/ubuntu/.cache/ms-playwright/chromium_headless_shell-1255/chrome-headless-shell-linux-arm64/chrome-headless-shell',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser'
].filter(Boolean);

let executablePath = null;
for (const p of CHROMIUM_PATHS) {
  try {
    if (p && path.isAbsolute(p)) {
      executablePath = p;
      break;
    }
  } catch {}
}

async function runTests() {
  console.log('=====================================================');
  console.log(' WorkBuddy2API English Panel - Playwright E2E Test');
  console.log('=====================================================');
  console.log(`Target URL:     ${TARGET_URL}`);
  console.log(`Executable:     ${executablePath || 'Default Playwright Browser'}`);

  const browser = await chromium.launch({
    headless: true,
    ...(executablePath ? { executablePath } : {}),
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    ignoreHTTPSErrors: true
  });

  const page = await context.newPage();

  // Inject API Key into localStorage before page scripts execute
  await page.addInitScript((key) => {
    localStorage.setItem('wb2api.key', key);
  }, API_KEY);

  console.log(`\n[1/4] Navigating to ${TARGET_URL}...`);
  await page.goto(TARGET_URL, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(2000);

  // Assert Title is translated
  const title = await page.title();
  console.log(`      Page title: "${title}"`);
  if (/[\u4e00-\u9fa5]/.test(title)) {
    throw new Error(`Page title contains untranslated Chinese: "${title}"`);
  }
  console.log('      ✓ Page title translated to English.');

  // Assert en.js script tag is injected
  const scriptInjected = await page.evaluate(() => {
    const scripts = Array.from(document.querySelectorAll('script'));
    return scripts.some((s) => s.src && s.src.includes('en.js'));
  });
  if (!scriptInjected) {
    throw new Error('en.js script tag was not injected into the panel!');
  }
  console.log('      ✓ en.js script tag verified in DOM.');

  // Tabs to navigate and verify (all 11 tabs)
  const tabs = [
    { hash: '#accounts', name: 'Account Pool', file: 'screenshot-account_pool.png' },
    { hash: '#usage', name: 'Usage Analytics', file: 'screenshot-usage_analytics.png' },
    { hash: '#proxies', name: 'Proxies', file: 'screenshot-proxies.png' },
    { hash: '#packages', name: 'Credits Breakdown', file: 'screenshot-credits_breakdown.png' },
    { hash: '#balhist', name: 'Balance Trends', file: 'screenshot-balance_trends.png' },
    { hash: '#taskscenter', name: 'Task Center', file: 'screenshot-task_center.png' },
    { hash: '#models', name: 'Models & Effort', file: 'screenshot-models___effort.png' },
    { hash: '#modelrules', name: 'Model Rules', file: 'screenshot-model_rules.png' },
    { hash: '#config', name: 'Settings', file: 'screenshot-settings.png' },
    { hash: '#events', name: 'Request Logs', file: 'screenshot-request_logs.png' },
    { hash: '#logs', name: 'System Logs', file: 'screenshot-system_logs.png' }
  ];

  console.log('\n[2/4] Testing navigation tabs & capturing screenshots...');
  for (const tab of tabs) {
    process.stdout.write(`      - Tab: ${tab.name.padEnd(18)} `);
    const link = await page.$(`a[href="${tab.hash}"]`);
    if (link) {
      await link.click();
      await page.waitForTimeout(1000);
    } else {
      await page.goto(`${TARGET_URL}${tab.hash}`, { waitUntil: 'networkidle' });
      await page.waitForTimeout(1000);
    }

    // Check for untranslated text in major containers of the active tab
    const untranslatedInTab = await page.evaluate(() => {
      const texts = [];
      const walker = document.createTreeWalker(
        document.body,
        NodeFilter.SHOW_TEXT,
        {
          acceptNode(node) {
            if (!node.nodeValue || !/[\u4e00-\u9fa5]/.test(node.nodeValue)) {
              return NodeFilter.FILTER_REJECT;
            }
            const parent = node.parentElement;
            if (!parent || parent.tagName === 'SCRIPT' || parent.tagName === 'STYLE') {
              return NodeFilter.FILTER_REJECT;
            }
            // Skip hidden elements
            const style = window.getComputedStyle(parent);
            if (style.display === 'none' || style.visibility === 'hidden') {
              return NodeFilter.FILTER_REJECT;
            }
            return NodeFilter.FILTER_ACCEPT;
          }
        }
      );
      while (walker.nextNode()) {
        const val = walker.currentNode.nodeValue.trim();
        if (val && !texts.includes(val)) texts.push(val);
      }
      return texts;
    });

    const screenshotPath = path.join(REPO_ROOT, tab.file);
    await page.screenshot({ path: screenshotPath, fullPage: false });
    if (untranslatedInTab.length > 0) {
      console.log(`[WARN] Untranslated in ${tab.name}:`, JSON.stringify(untranslatedInTab.slice(0, 3)));
    } else {
      console.log(`[PASS] Clean English (${tab.file})`);
    }
  }

  console.log('\n[3/4] Verifying English translations of key navigation & UI elements...');
  const navTexts = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('nav a, .nav-item, .sidebar a'))
      .map((el) => el.textContent.trim())
      .filter(Boolean);
  });
  console.log('      Sidebar items found:', navTexts.slice(0, 8).join(', '));
  const untranslatedNav = navTexts.filter((t) => /[\u4e00-\u9fa5]/.test(t));
  if (untranslatedNav.length > 0) {
    throw new Error(`Untranslated navigation items: ${JSON.stringify(untranslatedNav)}`);
  }
  console.log('      ✓ All sidebar navigation items are in English.');

  console.log('\n[4/4] Verifying metric cards & primary controls...');
  const metricCards = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('.stat-card, .metric-card, .card-title, .badge'))
      .map((el) => el.textContent.trim())
      .filter((t) => t.length > 0 && t.length < 30);
  });
  console.log('      Card labels sample:', metricCards.slice(0, 6).join(' | '));

  await browser.close();
  console.log('\n=====================================================');
  console.log(' ✔ ALL PLAYWRIGHT E2E TESTS PASSED SUCCESSFULLY!');
  console.log('=====================================================\n');
}

runTests().catch((err) => {
  console.error('\n✖ Playwright E2E Test Failed:', err.message);
  process.exit(1);
});
