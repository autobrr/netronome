#!/usr/bin/env node

import sharp from 'sharp';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const LOGO_PATH = path.join(__dirname, '../src/assets/logo.png');
const PUBLIC_DIR = path.join(__dirname, '../public');

const sizes = [
  { size: 192, name: 'pwa-192x192.png' },
  { size: 512, name: 'pwa-512x512.png' },
  { size: 180, name: 'apple-touch-icon.png' },
  { size: 64, name: 'pwa-64x64.png' },
  { size: 32, name: 'favicon-32x32.png' },
  { size: 16, name: 'favicon-16x16.png' }
];

async function generateIcons() {
  console.log('Generating PWA icons...');
  
  // Create public directory if it doesn't exist
  if (!fs.existsSync(PUBLIC_DIR)) {
    fs.mkdirSync(PUBLIC_DIR, { recursive: true });
  }

  for (const { size, name } of sizes) {
    try {
      await sharp(LOGO_PATH)
        .resize(size, size, {
          fit: 'contain',
          background: { r: 255, g: 255, b: 255, alpha: 0 }
        })
        .png()
        .toFile(path.join(PUBLIC_DIR, name));
      
      console.log(`✓ Generated ${name} (${size}x${size})`);
    } catch (error) {
      console.error(`✗ Failed to generate ${name}:`, error.message);
    }
  }

  // Generate maskable icon with padding
  try {
    await sharp(LOGO_PATH)
      .resize(512, 512, {
        fit: 'contain',
        background: { r: 255, g: 255, b: 255, alpha: 1 }
      })
      .extend({
        top: 50,
        bottom: 50,
        left: 50,
        right: 50,
        background: { r: 255, g: 255, b: 255, alpha: 1 }
      })
      .resize(512, 512)
      .png()
      .toFile(path.join(PUBLIC_DIR, 'pwa-maskable-512x512.png'));
    
    console.log('✓ Generated pwa-maskable-512x512.png');
  } catch (error) {
    console.error('✗ Failed to generate maskable icon:', error.message);
  }

  console.log('\nIcon generation complete!');
}

generateIcons().catch(console.error);