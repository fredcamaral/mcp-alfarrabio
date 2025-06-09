# Static Assets for Swagger UI

This directory contains static assets for the Swagger UI integration.

In production, you would include actual Swagger UI assets here:
- CSS files
- JavaScript files  
- Fonts
- Images

For now, we're using CDN-hosted assets in the HTML template.

## Assets to Include

- `swagger-ui-bundle.js`
- `swagger-ui-standalone-preset.js`
- `swagger-ui.css`
- `favicon-*.png` files

## CDN Alternative

Currently using CDN links for simplicity:
- https://unpkg.com/swagger-ui-dist@5.10.5/

For production deployments, consider downloading and hosting these assets locally for better performance and reliability.