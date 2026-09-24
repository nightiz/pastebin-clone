const express = require('express');
const router = express.Router();
const apiClient = require('../lib/apiClient');

// Helper to render friendly error pages
function handleRouteError(err, res, customTitle = 'Error') {
  if (err.isConnectionError) {
    return res.status(503).render('error', {
      title: 'Backend Offline',
      statusCode: 503,
      message: 'The Go backend server is currently unreachable.',
      details: 'Ensure the backend is running at ' + (process.env.BACKEND_API_URL || 'http://localhost:8080') + ' via "go run ./cmd/server" in the backend directory.',
      isConnectionError: true
    });
  }

  const status = err.status || 500;
  let userMessage = err.message || 'Something went wrong.';
  if (status === 404) {
    userMessage = 'This paste either does not exist, has expired, or was already burned after reading.';
  }

  res.status(status).render('error', {
    title: customTitle,
    statusCode: status,
    message: userMessage,
    details: err.code ? `Error code: ${err.code}` : null,
    isConnectionError: false
  });
}

// GET / - Paste creation form
router.get('/', (req, res) => {
  res.render('index', {
    title: 'New Paste',
    formData: {},
    error: null,
    deleted: req.query.deleted === '1'
  });
});

// POST /pastes - Submit new paste
router.post('/pastes', async (req, res) => {
  const { content, title, language, expires_in_seconds, burn_after_read } = req.body;

  if (!content || !content.trim()) {
    return res.status(400).render('index', {
      title: 'New Paste',
      formData: req.body,
      error: 'Paste content cannot be empty.',
      deleted: false
    });
  }

  try {
    const result = await apiClient.createPaste({
      content,
      title,
      language,
      expiresInSeconds: expires_in_seconds,
      burnAfterRead: burn_after_read === 'on' || burn_after_read === 'true' || burn_after_read === true
    });

    // Pass the edit token in query params so the creator can save or delete it
    res.redirect(`/pastes/${result.id}?token=${encodeURIComponent(result.edit_token)}&created=1`);
  } catch (err) {
    if (err.isConnectionError) {
      return handleRouteError(err, res);
    }
    res.status(err.status || 400).render('index', {
      title: 'New Paste',
      formData: req.body,
      error: err.message || 'Failed to create paste',
      deleted: false
    });
  }
});

// GET /pastes/:id - View paste
router.get('/pastes/:id', async (req, res) => {
  const { id } = req.params;
  const token = req.query.token || '';
  const created = req.query.created === '1';

  try {
    const paste = await apiClient.getPaste(id);
    res.render('paste', {
      title: paste.title ? `${paste.title} — Paste` : `Paste ${paste.id}`,
      paste,
      editToken: token,
      justCreated: created,
      error: null
    });
  } catch (err) {
    handleRouteError(err, res, 'Paste Not Found');
  }
});

// GET /pastes/:id/raw - Raw text content
router.get('/pastes/:id/raw', async (req, res) => {
  const { id } = req.params;

  try {
    const rawContent = await apiClient.getPasteRaw(id);
    res.setHeader('Content-Type', 'text/plain; charset=utf-8');
    res.send(rawContent);
  } catch (err) {
    if (err.isConnectionError) {
      res.setHeader('Content-Type', 'text/plain; charset=utf-8');
      return res.status(503).send('Backend service unavailable');
    }
    res.setHeader('Content-Type', 'text/plain; charset=utf-8');
    res.status(err.status || 404).send(err.message || 'Paste not found');
  }
});

// POST /pastes/:id/delete - Delete a paste
router.post('/pastes/:id/delete', async (req, res) => {
  const { id } = req.params;
  const editToken = req.body.edit_token || req.query.token;

  if (!editToken) {
    return res.status(400).render('error', {
      title: 'Unauthorized',
      statusCode: 400,
      message: 'Edit token is required to delete this paste.',
      details: null,
      isConnectionError: false
    });
  }

  try {
    await apiClient.deletePaste(id, editToken);
    res.redirect('/?deleted=1');
  } catch (err) {
    handleRouteError(err, res, 'Deletion Failed');
  }
});

module.exports = router;
