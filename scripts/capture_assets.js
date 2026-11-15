const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const BIN_PATH = path.join(__dirname, '..', 'bin', 'claimguard.exe');
const CONFIG_PATH = path.join(__dirname, '..', 'examples', 'claimguard.yaml');
const IMAGES_DIR = path.join(__dirname, '..', 'docs', 'images');
const FRAMES_DIR = path.join(__dirname, 'frames');

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function clickButtonWithText(page, text) {
  const buttons = await page.$$('button');
  for (const b of buttons) {
    const txt = await page.evaluate(el => el.textContent, b);
    if (txt && txt.includes(text)) {
      await b.click();
      return true;
    }
  }
  return false;
}

async function run() {
  if (!fs.existsSync(IMAGES_DIR)) {
    fs.mkdirSync(IMAGES_DIR, { recursive: true });
  }
  if (fs.existsSync(FRAMES_DIR)) {
    fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(FRAMES_DIR, { recursive: true });

  console.log('1. Starting claimguard server...');
  const server = spawn(BIN_PATH, ['serve', '--config', CONFIG_PATH], {
    stdio: 'inherit',
    cwd: path.join(__dirname, '..'),
  });

  // Give server time to initialize
  await sleep(2500);

  console.log('2. Launching headless Chrome...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-gpu', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('3. Navigating to http://localhost:9090...');
  await page.goto('http://localhost:9090', { waitUntil: 'domcontentloaded' });
  await sleep(2000);

  // Trigger automated simulation events to populate metrics and audit log
  console.log('4. Generating initial authentic compliance audit events...');
  try {
    // Preset 1: Return Policy
    await clickButtonWithText(page, 'Return Policy');
    await sleep(300);
    await clickButtonWithText(page, 'Send to Proxy');
    await sleep(3000);

    // Preset 2: Enterprise Pricing
    await clickButtonWithText(page, 'Enterprise Pricing');
    await sleep(300);
    await clickButtonWithText(page, 'Send to Proxy');
    await sleep(3000);

    // Preset 3: SLA Uptime
    await clickButtonWithText(page, 'SLA Uptime');
    await sleep(300);
    await clickButtonWithText(page, 'Send to Proxy');
    await sleep(3000);

    // Return to "Return Policy" preset for pristine side-by-side display
    await clickButtonWithText(page, 'Return Policy');
    await sleep(300);
    await clickButtonWithText(page, 'Send to Proxy');
    await sleep(3000);
  } catch (err) {
    console.error('Error populating presets:', err);
  }

  // 1. Audit Cockpit (Desktop Overview)
  console.log('5. Capturing dashboard_cockpit.png...');
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'dashboard_cockpit.png'),
    fullPage: false,
  });

  // 2. Stream Auditor Inline Diff
  console.log('6. Switching to Inline Diff mode and capturing stream_auditor_diff.png...');
  await clickButtonWithText(page, 'Inline Diff');
  await sleep(600);
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'stream_auditor_diff.png'),
    fullPage: false,
  });

  // Switch back to Side-by-Side
  await clickButtonWithText(page, 'Side-by-Side');
  await sleep(300);

  // 3. Ground Truth Registry Tab
  console.log('7. Switching to Ground Truth Registry tab...');
  await clickButtonWithText(page, 'Ground Truth Registry');
  await sleep(800);
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'ground_truth_registry.png'),
    fullPage: false,
  });

  // 4. Compliance Ledger Tab
  console.log('8. Switching to Compliance Ledger tab...');
  await clickButtonWithText(page, 'Compliance Ledger');
  await sleep(800);
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'compliance_ledger.png'),
    fullPage: false,
  });

  // 5. Red-Team Benchmark Tab
  console.log('9. Switching to Red-Team Benchmark tab and executing scan...');
  await clickButtonWithText(page, 'Red-Team Benchmark');
  await sleep(800);
  await clickButtonWithText(page, 'Run Red-Team Scan');
  console.log('   Executing benchmark suite...');
  await sleep(3000);
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'redteam_benchmark.png'),
    fullPage: false,
  });

  // 6. Mobile Responsiveness Screenshot (390x844 - iPhone 14/15)
  console.log('10. Capturing mobile_cockpit.png on 390x844 viewport...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });
  await clickButtonWithText(page, 'Audit Cockpit');
  await sleep(600);
  await page.screenshot({
    path: path.join(IMAGES_DIR, 'mobile_cockpit.png'),
    fullPage: false,
  });

  // 7. Record Animated Demo GIF
  console.log('11. Recording frame sequence for claimguard_demo.gif...');
  await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1 });
  await clickButtonWithText(page, 'Audit Cockpit');
  await sleep(400);

  // Select Loan APR
  await clickButtonWithText(page, 'Loan APR');
  await sleep(300);

  // Trigger Send and capture rapid frames
  await clickButtonWithText(page, 'Send to Proxy');

  // Capture 30 frames across the streaming duration
  for (let i = 0; i < 30; i++) {
    const framePath = path.join(FRAMES_DIR, `frame_${String(i).padStart(4, '0')}.png`);
    await page.screenshot({ path: framePath, fullPage: false });
    await sleep(100);
  }

  await browser.close();

  // Terminate server
  console.log('12. Shutting down server...');
  server.kill('SIGTERM');
  await sleep(1000);

  // Compile frames into GIF using ffmpeg
  console.log('13. Compiling claimguard_demo.gif using ffmpeg...');
  const gifPath = path.join(IMAGES_DIR, 'claimguard_demo.gif');
  const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${path.join(FRAMES_DIR, 'frame_%04d.png')}" -vf "fps=10,scale=1000:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer" "${gifPath}"`;
  
  execSync(ffmpegCmd, { stdio: 'inherit' });
  console.log('✓ Successfully generated claimguard_demo.gif');

  // Clean up frames directory
  fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  console.log('✓ Assets capture completed successfully!');
}

run().catch((err) => {
  console.error('Asset capture failed:', err);
  process.exit(1);
});
