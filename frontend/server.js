const app = require('./app');

const PORT = parseInt(process.env.PORT, 10) || 3000;

const server = app.listen(PORT, () => {
  console.log(`[Frontend] Server listening on http://localhost:${PORT}`);
  console.log(`[Frontend] Backend API URL: ${process.env.BACKEND_API_URL || 'http://localhost:8080'}`);
});

process.on('SIGINT', () => {
  console.log('\n[Frontend] Shutting down gracefully...');
  server.close(() => {
    process.exit(0);
  });
});
