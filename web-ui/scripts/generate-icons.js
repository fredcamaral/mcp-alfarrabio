const fs = require('fs');
const path = require('path');

// Create a simple 1x1 pixel transparent PNG
const sizes = [72, 96, 128, 144, 152, 192, 384, 512];

// Base64 encoded 1x1 transparent PNG
const transparentPNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
  'base64'
);

// Create icons directory if it doesn't exist
const iconsDir = path.join(__dirname, '../public/icons');
if (!fs.existsSync(iconsDir)) {
  fs.mkdirSync(iconsDir, { recursive: true });
}

// Generate placeholder icons
sizes.forEach(size => {
  const filename = path.join(iconsDir, `icon-${size}x${size}.png`);
  fs.writeFileSync(filename, transparentPNG);
  console.log(`Created ${filename}`);
});

// Also create the shortcut icons
const shortcuts = ['search-96x96.png', 'performance-96x96.png'];
shortcuts.forEach(name => {
  const filename = path.join(iconsDir, name);
  fs.writeFileSync(filename, transparentPNG);
  console.log(`Created ${filename}`);
});

console.log('All placeholder icons created successfully!');