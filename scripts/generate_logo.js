const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <linearGradient id="amber-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fef08a"/>
      <stop offset="30%" stop-color="#f59e0b"/>
      <stop offset="100%" stop-color="#d97706"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.16" />
    </filter>

    <filter id="eye-glow" x="-30%" y="-30%" width="160%" height="160%">
      <feDropShadow dx="0" dy="0" stdDeviation="6" flood-color="#f59e0b" flood-opacity="0.8" />
    </filter>

    <filter id="core-glow" x="-20%" y="-20%" width="140%" height="140%">
      <feDropShadow dx="0" dy="0" stdDeviation="8" flood-color="#00f5ff" flood-opacity="0.7" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-390
        338,-195
        338,195
        0,390
        -338,195
        -338,-195
      " fill="none" stroke="#0f172a" stroke-width="38" stroke-linejoin="round" />

      <polygon points="
        0,-355
        307,-177
        307,177
        0,355
        -307,177
        -307,-177
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.45" stroke-dasharray="16, 12" />

      <!-- Gateway Corner Telemetry Nodes -->
      <circle cx="0" cy="-390" r="10" fill="#00f5ff" />
      <circle cx="338" cy="-195" r="10" fill="#00f5ff" />
      <circle cx="338" cy="195" r="10" fill="#00f5ff" />
      <circle cx="0" cy="390" r="10" fill="#00f5ff" />
      <circle cx="-338" cy="195" r="10" fill="#00f5ff" />
      <circle cx="-338" cy="-195" r="10" fill="#00f5ff" />

      <!-- 100% Watertight Solid Silhouette of the Aegis Falcon -->
      <path d="
        M 0 -290
        L 32 -260 L 45 -290 L 80 -230 L 72 -200 L 140 -150 L 175 -80 L 160 -10 L 220 30 L 280 90 L 270 180 L 230 260 L 160 325 L 90 290 L 50 320 L 0 355
        L -50 320 L -90 290 L -160 325 L -230 260 L -270 180 L -280 90 L -220 30 L -160 -10 L -175 -80 L -140 -150 L -72 -200 L -80 -230 L -45 -290 L -32 -260
        Z
      " fill="#090d16" stroke="#0f172a" stroke-width="2" />

      <!-- Wing Flight Feathers & Armor Plates (Faceted Volumetric Layers) -->
      <!-- Left Outer Wing -->
      <polygon points="-160,-10 -220,30 -280,90 -210,110 -140,50" fill="#1e293b" stroke="#0f172a" stroke-width="2" />
      <polygon points="-280,90 -270,180 -195,175 -210,110" fill="#0f172a" stroke="#090d16" stroke-width="2" />
      <polygon points="-270,180 -230,260 -160,245 -195,175" fill="#1e293b" stroke="#0f172a" stroke-width="2" />
      <polygon points="-230,260 -160,325 -105,290 -160,245" fill="#0b1120" stroke="#0f172a" stroke-width="2" />

      <!-- Right Outer Wing (Mirrored) -->
      <polygon points="160,-10 220,30 280,90 210,110 140,50" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="280,90 270,180 195,175 210,110" fill="#1e293b" stroke="#090d16" stroke-width="2" />
      <polygon points="270,180 230,260 160,245 195,175" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="230,260 160,325 105,290 160,245" fill="#1e293b" stroke="#0f172a" stroke-width="2" />

      <!-- Wing Seam Telemetry Accents -->
      <line x1="-280" y1="90" x2="-270" y2="180" stroke="#00f5ff" stroke-width="3" opacity="0.8" />
      <line x1="280" y1="90" x2="270" y2="180" stroke="#00f5ff" stroke-width="3" opacity="0.8" />
      <line x1="-270" y1="180" x2="-230" y2="260" stroke="#00f5ff" stroke-width="2.5" opacity="0.5" />
      <line x1="270" y1="180" x2="230" y2="260" stroke="#00f5ff" stroke-width="2.5" opacity="0.5" />

      <!-- Lower Flanks & Tail Feathers -->
      <polygon points="-130,130 -195,175 -160,245 -90,250 -60,210 -60,150" fill="#182235" stroke="#0f172a" stroke-width="2" />
      <polygon points="130,130 195,175 160,245 90,250 60,210 60,150" fill="#24334a" stroke="#0f172a" stroke-width="2" />
      <polygon points="-60,210 -90,250 -105,290 -50,320 0,260" fill="#0f172a" stroke="#090d16" stroke-width="2" />
      <polygon points="60,210 90,250 105,290 50,320 0,260" fill="#1e293b" stroke="#090d16" stroke-width="2" />
      <polygon points="0,260 50,320 0,355 -50,320" fill="#0b1120" stroke="#00f5ff" stroke-width="2" opacity="0.9" />

      <!-- Mantle & Breast Armor Plates -->
      <polygon points="-50,60 -140,50 -210,110 -130,130 -60,150 0,115" fill="#1e293b" stroke="#0f172a" stroke-width="2" />
      <polygon points="50,60 140,50 210,110 130,130 60,150 0,115" fill="#334155" stroke="#0f172a" stroke-width="2" />

      <!-- Falcon Sharp Swept Crest Tuft Facets -->
      <polygon points="0,-290 32,-260 0,-215 -32,-260" fill="#475569" stroke="#0f172a" stroke-width="2" />
      <!-- Crest Ear-Tuft Horns (Aerodynamic Raptor Horns) -->
      <polygon points="32,-260 45,-290 80,-230 45,-215" fill="#64748b" stroke="#0f172a" stroke-width="2" />
      <polygon points="-32,-260 -45,-290 -80,-230 -45,-215" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="45,-215 80,-230 72,-200 40,-175" fill="#475569" stroke="#0f172a" stroke-width="2" />
      <polygon points="-45,-215 -80,-230 -72,-200 -40,-175" fill="#1e293b" stroke="#0f172a" stroke-width="2" />

      <!-- Forehead & Eyebrows (Supraorbital Ridges) -->
      <polygon points="0,-215 45,-215 40,-175 0,-130" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="0,-215 -45,-215 -40,-175 0,-130" fill="#1e293b" stroke="#0f172a" stroke-width="2" />

      <!-- Fierce Angular Brow Plates -->
      <polygon points="0,-130 65,-145 130,-125 70,-100 0,-85" fill="#475569" stroke="#0f172a" stroke-width="2" />
      <polygon points="0,-130 -65,-145 -130,-125 -70,-100 0,-85" fill="#1e293b" stroke="#0f172a" stroke-width="2" />
      <polygon points="130,-125 140,-150 72,-200 40,-175 65,-145" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="-130,-125 -140,-150 -72,-200 -40,-175 -65,-145" fill="#182235" stroke="#0f172a" stroke-width="2" />

      <!-- Cheeks & Malar Low-Poly Facets -->
      <polygon points="-70,-100 -140,-115 -175,-80 -115,-45 -45,-80" fill="#182235" stroke="#0f172a" stroke-width="2" />
      <polygon points="70,-100 140,-115 175,-80 115,-45 45,-80" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="-115,-45 -175,-80 -160,-10 -90,0" fill="#0f172a" stroke="#090d16" stroke-width="2" />
      <polygon points="115,-45 175,-80 160,-10 90,0" fill="#1e293b" stroke="#090d16" stroke-width="2" />

      <!-- Predator Optics (Almond-Slanted Falcon Eyes with Upward Cant & Vivid Luster) -->
      <!-- Left Eye Socket -->
      <polygon points="-38,-82 -110,-95 -145,-120 -65,-106" fill="#090d16" stroke="#0f172a" stroke-width="2" />
      <!-- Left Eye Iris (Vivid Amber Gold) -->
      <polygon points="-46,-84 -102,-95 -136,-115 -68,-103" fill="url(#amber-glow)" filter="url(#eye-glow)" />
      <!-- Left Eye Sharp Slit Pupil & Cyan Telemetry Spark -->
      <polygon points="-88,-100 -78,-94 -92,-97" fill="#090d16" />
      <circle cx="-84" cy="-97" r="2.5" fill="#00f5ff" />

      <!-- Right Eye Socket (Mirrored) -->
      <polygon points="38,-82 110,-95 145,-120 65,-106" fill="#090d16" stroke="#0f172a" stroke-width="2" />
      <!-- Right Eye Iris (Vivid Amber Gold) -->
      <polygon points="46,-84 102,-95 136,-115 68,-103" fill="url(#amber-glow)" filter="url(#eye-glow)" />
      <!-- Right Eye Sharp Slit Pupil & Cyan Telemetry Spark -->
      <polygon points="88,-100 78,-94 92,-97" fill="#090d16" />
      <circle cx="84" cy="-97" r="2.5" fill="#00f5ff" />

      <!-- Aerodynamic Hooked Raptor Beak & Cere -->
      <polygon points="0,-85 35,-55 0,-45 -35,-55" fill="#475569" stroke="#0f172a" stroke-width="2" />
      <polygon points="0,-45 28,-10 0,35 -28,-10" fill="#94a3b8" stroke="#0f172a" stroke-width="2" />
      <polygon points="0,-45 -35,-55 -40,-15 -28,-10" fill="#334155" stroke="#0f172a" stroke-width="2" />
      <polygon points="0,-45 35,-55 40,-15 28,-10" fill="#64748b" stroke="#0f172a" stroke-width="2" />
      <!-- Hooked Tip -->
      <polygon points="0,35 18,10 0,55 -18,10" fill="#0f172a" stroke="#00f5ff" stroke-width="2" />

      <!-- Throat Chevrons -->
      <polygon points="0,55 50,60 0,115 -50,60" fill="#182235" stroke="#0f172a" stroke-width="2" />

      <!-- Central Hexagonal Guard Shield (The ClaimGuard Airlock Core) -->
      <polygon points="0,130 55,160 55,220 0,250 -55,220 -55,160" fill="#090d16" stroke="#0f172a" stroke-width="4" />
      <polygon points="0,145 42,168 42,212 0,235 -42,212 -42,168" fill="none" stroke="url(#cyan-glow)" stroke-width="4" filter="url(#core-glow)" />
      <polygon points="0,160 28,175 28,205 0,220 -28,205 -28,175" fill="#0f172a" stroke="#00f5ff" stroke-width="2" />

      <!-- Micro Verification Checkmark in Shield Core -->
      <path d="M -10 190 L -3 197 L 12 182" fill="none" stroke="#00f5ff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />

    </g>
  </g>
</svg>`;
}

async function renderLogo(outputDir) {
  const svg = buildLogoSvg();
  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg);
  const resvg = new Resvg(svg, { fitTo: { mode: 'width', value: 1024 } });
  fs.writeFileSync(pngPath, resvg.render().asPng());
  console.log('✓ Successfully rendered logo.svg and logo.png at 1024x1024');
}

renderLogo(path.join(__dirname, '..', 'docs', 'images'));
