const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const BIN_PATH = path.join(__dirname, '..', 'bin', 'claimguard.exe');
const CONFIG_PATH = path.join(__dirname, '..', 'examples', 'claimguard.yaml');
const IMAGES_DIR = path.join(__dirname, '..', 'docs', 'images');
const FRAMES_DIR = path.join(__dirname, 'gif_frames');

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

async function record() {
  if (fs.existsSync(FRAMES_DIR)) {
    fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(FRAMES_DIR, { recursive: true });

  console.log('1. Launching claimguard server...');
  const server = spawn(BIN_PATH, ['serve', '--config', CONFIG_PATH], {
    stdio: 'ignore',
    cwd: path.join(__dirname, '..'),
  });

  await sleep(2500);

  console.log('2. Launching headless browser for GIF recording...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-gpu', '--window-size=1200,750'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1 });

  console.log('3. Loading cockpit...');
  await page.goto('http://localhost:9090', { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  console.log('4. Selecting Return Policy preset...');
  await clickButtonWithText(page, 'Return Policy');
  await sleep(400);

  let frameIdx = 0;
  const takeFrame = async () => {
    const framePath = path.join(FRAMES_DIR, `frame_${String(frameIdx++).padStart(4, '0')}.png`);
    await page.screenshot({ path: framePath, fullPage: false });
  };

  // 1. Initial 5 frames showing prompt ready
  console.log('5. Capturing initial frames...');
  for (let i = 0; i < 5; i++) {
    await takeFrame();
    await sleep(80);
  }

  // 2. Click Send to Proxy
  console.log('6. Clicking Send to Proxy...');
  await clickButtonWithText(page, 'Send to Proxy');

  // 3. Capture 45 frames during streaming
  console.log('7. Capturing active streaming frames...');
  for (let i = 0; i < 45; i++) {
    await takeFrame();
    await sleep(80);
  }

  // 4. Wait for stream to finish completely if needed
  await sleep(1500);

  // 5. Capture 15 final frames showing complete rewritten sentence and breakdown
  console.log('8. Capturing completed state frames...');
  for (let i = 0; i < 15; i++) {
    await takeFrame();
    await sleep(80);
  }

  await browser.close();

  // Terminate server safely on Windows
  console.log('9. Closing server...');
  try {
    execSync('taskkill /F /IM claimguard.exe /T', { stdio: 'ignore' });
  } catch (e) {}

  await sleep(1000);

  console.log('10. Assembling demo GIF with ffmpeg...');
  const gifPath = path.join(IMAGES_DIR, 'claimguard_demo.gif');
  const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${path.join(FRAMES_DIR, 'frame_%04d.png')}" -vf "fps=10,scale=1000:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer" "${gifPath}"`;
  execSync(ffmpegCmd, { stdio: 'inherit' });

  // Clean up frames directory
  fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  console.log('✓ claimguard_demo.gif successfully recorded and optimized!');
}

record().catch((err) => {
  console.error('Recording failed:', err);
  try {
    execSync('taskkill /F /IM claimguard.exe /T', { stdio: 'ignore' });
  } catch (e) {}
  process.exit(1);
});
