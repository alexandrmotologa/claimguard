const { spawn, execSync } = require('child_process');
const path = require('path');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const BIN_PATH = path.join(__dirname, '..', 'bin', 'claimguard.exe');
const CONFIG_PATH = path.join(__dirname, '..', 'examples', 'claimguard.yaml');
const IMAGES_DIR = path.join(__dirname, '..', 'docs', 'images');

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function run() {
  console.log('1. Starting server...');
  const server = spawn(BIN_PATH, ['serve', '--config', CONFIG_PATH], {
    stdio: 'ignore',
    cwd: path.join(__dirname, '..'),
  });

  await sleep(2500);

  console.log('2. Launching browser on iPhone 14/15 viewport (390x844)...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-gpu'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });

  await page.goto('http://localhost:9090', { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  // Send one test prompt so metrics are populated
  const buttons = await page.$$('button');
  for (const b of buttons) {
    const txt = await page.evaluate(el => el.textContent, b);
    if (txt && txt.includes('Send to Proxy')) {
      await b.click();
      break;
    }
  }
  await sleep(2500);

  // Scroll back to top so header and metrics are fully visible
  await page.evaluate(() => window.scrollTo(0, 0));
  await sleep(500);

  console.log('3. Capturing mobile_cockpit.png...');
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'mobile_cockpit.png'),
    fullPage: false,
  });

  await browser.close();

  try {
    execSync('taskkill /F /IM claimguard.exe /T', { stdio: 'ignore' });
  } catch (e) {}

  console.log('✓ Successfully recaptured mobile_cockpit.png at top scroll!');
}

run().catch((err) => {
  console.error(err);
  try {
    execSync('taskkill /F /IM claimguard.exe /T', { stdio: 'ignore' });
  } catch (e) {}
  process.exit(1);
});
