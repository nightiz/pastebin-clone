const path = require('path');
const express = require('express');
const dotenv = require('dotenv');

dotenv.config();

const pasteRoutes = require('./routes/pastes');

const app = express();

// View engine setup
app.set('view engine', 'ejs');
app.set('views', path.join(__dirname, 'views'));

// Body parsing middleware
app.use(express.urlencoded({ extended: true, limit: '10mb' }));
app.use(express.json({ limit: '10mb' }));

// Static assets (compiled CSS, client JS)
app.use(express.static(path.join(__dirname, 'public')));

// Mount routes
app.use('/', pasteRoutes);

// Fallback 404 handler
app.use((req, res) => {
  res.status(404).render('error', {
    title: 'Page Not Found',
    statusCode: 404,
    message: 'The requested page does not exist.',
    details: `Path: ${req.path}`,
    isConnectionError: false
  });
});

module.exports = app;
