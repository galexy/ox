/**
 * Browser automation for ox login demo
 *
 * This script automates the browser login flow for the ox CLI.
 * It navigates to the device code verification URL and fills in credentials.
 *
 * Usage:
 *   DEMO_EMAIL=you@yourcompany.com DEMO_PASSWORD=secret ts-node login.ts <verification_url>
 *
 * Environment variables:
 *   DEMO_EMAIL     - Login email
 *   DEMO_PASSWORD  - Login password
 *   HEADLESS       - Set to "false" to show browser (default: true)
 *   DEBUG          - Set to "1" for verbose logging
 */

import { chromium, Browser, Page, BrowserContext } from '@playwright/test';

const DEBUG = process.env.DEBUG === '1';

function debug(...args: unknown[]): void {
  if (DEBUG) {
    console.log('[demo-login]', ...args);
  }
}

// selectors for SageOx auth page (Better Auth)
const EMAIL_SELECTORS = [
  'input[type="email"]',
  'input[name="email"]',
  'input[id="email"]',
];

const PASSWORD_SELECTORS = [
  'input[type="password"]',
  'input[name="password"]',
];

const SUBMIT_SELECTORS = [
  'button[type="submit"]',
  'button:has-text("Sign in")',
  'button:has-text("Log in")',
  'button:has-text("Continue")',
];

async function findAndFill(page: Page, selectors: string[], value: string, fieldName: string): Promise<boolean> {
  for (const selector of selectors) {
    try {
      const el = await page.$(selector);
      if (el && await el.isVisible()) {
        await el.fill(value);
        debug(`filled ${fieldName} using: ${selector}`);
        return true;
      }
    } catch {
      // try next
    }
  }
  return false;
}

async function findAndClick(page: Page, selectors: string[]): Promise<boolean> {
  for (const selector of selectors) {
    try {
      const el = await page.$(selector);
      if (el && await el.isVisible()) {
        await el.click();
        debug(`clicked using: ${selector}`);
        return true;
      }
    } catch {
      // try next
    }
  }
  return false;
}

async function performLogin(url: string, email: string, password: string): Promise<void> {
  const headless = process.env.HEADLESS !== 'false';
  let browser: Browser | null = null;
  let context: BrowserContext | null = null;
  let page: Page | null = null;

  try {
    debug(`launching browser (headless: ${headless})`);
    browser = await chromium.launch({ headless });
    context = await browser.newContext();
    page = await context.newPage();

    debug(`navigating to: ${url}`);
    await page.goto(url, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    debug(`page loaded: ${page.url()}`);

    // fill email
    if (!await findAndFill(page, EMAIL_SELECTORS, email, 'email')) {
      throw new Error('Could not find email field');
    }

    // check if password is visible, or need to click continue first
    let passwordVisible = false;
    for (const sel of PASSWORD_SELECTORS) {
      const el = await page.$(sel);
      if (el && await el.isVisible()) {
        passwordVisible = true;
        break;
      }
    }

    if (!passwordVisible) {
      debug('clicking continue to reveal password field');
      await findAndClick(page, SUBMIT_SELECTORS);
      await page.waitForTimeout(2000);
    }

    // fill password
    if (!await findAndFill(page, PASSWORD_SELECTORS, password, 'password')) {
      throw new Error('Could not find password field');
    }

    // submit
    if (!await findAndClick(page, SUBMIT_SELECTORS)) {
      debug('no submit button, pressing Enter');
      await page.keyboard.press('Enter');
    }

    // wait for success (redirect away from signin)
    debug('waiting for login success...');
    await page.waitForURL(
      (u) => {
        const s = u.toString();
        return !s.includes('/signin') && !s.includes('/login') && !s.includes('/device');
      },
      { timeout: 30000 }
    );

    debug(`login successful, redirected to: ${page.url()}`);
    console.log('Login completed successfully');

  } finally {
    if (context) await context.close();
    if (browser) await browser.close();
  }
}

async function main(): Promise<void> {
  const url = process.argv[2];
  const email = process.env.DEMO_EMAIL;
  const password = process.env.DEMO_PASSWORD;

  if (!url) {
    console.error('Usage: ts-node login.ts <verification_url>');
    console.error('Environment: DEMO_EMAIL, DEMO_PASSWORD');
    process.exit(1);
  }

  if (!email || !password) {
    console.error('Error: DEMO_EMAIL and DEMO_PASSWORD must be set');
    process.exit(1);
  }

  debug(`logging in as: ${email}`);
  await performLogin(url, email, password);
}

main().catch((err) => {
  console.error('Login failed:', err.message);
  process.exit(1);
});
