// External Swagger UI initializer. Lives outside the HTML so the response
// body has no inline <script>, letting Lamp nginx keep `script-src 'self'`
// without `'unsafe-inline'`.
//
// Relative paths so the page works in two contexts:
//   - via Lamp proxy iframe (/api/hardware/docs) → fetches ./openapi.json
//     which resolves to /api/hardware/openapi.json
//   - direct LeLamp loopback (/docs via SSH tunnel) → fetches ./openapi.json
//     which resolves to /openapi.json on LeLamp
//
// FastAPI's `servers=[...]` list (set in lelamp/server.py) drives the base
// URL Swagger UI uses when the operator clicks "Try it out", so Execute
// reaches the proxied `/api/hardware/*` path in the browser context.
window.addEventListener("load", function () {
  window.ui = SwaggerUIBundle({
    url: "./openapi.json",
    dom_id: "#swagger-ui",
    layout: "BaseLayout",
    deepLinking: true,
    showExtensions: true,
    showCommonExtensions: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIBundle.SwaggerUIStandalonePreset,
    ],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
  });
});
