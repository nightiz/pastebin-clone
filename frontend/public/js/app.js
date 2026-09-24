// Copy text helper with visual feedback
function copyToClipboard(text, buttonElement) {
  if (!text) return;

  const originalContent = buttonElement ? buttonElement.innerHTML : null;

  function onSuccess() {
    if (buttonElement) {
      buttonElement.innerHTML = `
        <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
        <span class="text-emerald-300">Copied!</span>
      `;
      setTimeout(() => {
        buttonElement.innerHTML = originalContent;
      }, 2000);
    }
  }

  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(onSuccess).catch(() => fallbackCopy(text, onSuccess));
  } else {
    fallbackCopy(text, onSuccess);
  }
}

function fallbackCopy(text, cb) {
  const textArea = document.createElement('textarea');
  textArea.value = text;
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.select();
  try {
    document.execCommand('copy');
    if (cb) cb();
  } catch (err) {
    console.error('Fallback copy failed', err);
  }
  document.body.removeChild(textArea);
}

function copyPasteContent(buttonElement) {
  const codeEl = document.getElementById('pasteCode');
  if (codeEl) {
    copyToClipboard(codeEl.textContent, buttonElement);
  }
}

function toggleDeleteModal() {
  const modal = document.getElementById('deleteModal');
  if (modal) {
    modal.classList.toggle('hidden');
    if (!modal.classList.contains('hidden')) {
      const input = document.getElementById('deleteTokenInput');
      if (input && !input.value) {
        input.focus();
      }
    }
  }
}

// Textarea enhancements (line/char counts, tab indentation, Ctrl+Enter submit)
document.addEventListener('DOMContentLoaded', () => {
  const textarea = document.getElementById('content');
  const lineCountEl = document.getElementById('lineCount');
  const charCountEl = document.getElementById('charCount');
  const form = document.getElementById('pasteForm');

  if (textarea) {
    function updateCounts() {
      const val = textarea.value;
      const lines = val ? val.split('\n').length : 0;
      const chars = val ? val.length : 0;

      if (lineCountEl) {
        lineCountEl.textContent = `${lines} ${lines === 1 ? 'line' : 'lines'}`;
      }
      if (charCountEl) {
        charCountEl.textContent = `${chars} ${chars === 1 ? 'character' : 'characters'}`;
      }
    }

    textarea.addEventListener('input', updateCounts);
    updateCounts();

    // Tab key indentation support
    textarea.addEventListener('keydown', (e) => {
      if (e.key === 'Tab') {
        e.preventDefault();
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const value = textarea.value;

        // Insert 2 spaces for tab
        textarea.value = value.substring(0, start) + '  ' + value.substring(end);
        textarea.selectionStart = textarea.selectionEnd = start + 2;
        updateCounts();
      } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
        if (form) {
          form.requestSubmit();
        }
      }
    });
  }
});
