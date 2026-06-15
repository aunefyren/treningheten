/* ============================================================================
   TRModal — the single shared modal controller (see web/css/modal.css).
   ----------------------------------------------------------------------------
   Lazily injects one reusable panel into <body> and drives it. Every page that
   used to hand-render its own #myModal markup now shares this one component.

   API:
     TRModal.open({ eyebrow, title, body, onClose, variant })  -> show
     TRModal.setBody(html)                                     -> swap body only
     TRModal.close()                                           -> hide
     TRModal.isOpen()                                          -> bool

   Backward-compatible globals (so existing callers keep working):
     closeModal()              -> TRModal.close()
     toggleModal(html?)        -> html ? open({body:html}) : close()

   Behaviour: ESC + backdrop + close-button dismiss, body scroll-lock, focus
   restore, and graceful close when motion is reduced.
   ============================================================================ */

const TRModal = (function () {
    var root = null;
    var panel = null;
    var bodyEl = null;
    var titleEl = null;
    var eyebrowEl = null;
    var headEl = null;
    var onCloseCb = null;
    var lastFocus = null;

    var CLOSE_ICON =
        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">' +
        '<line x1="5" y1="5" x2="19" y2="19"></line><line x1="19" y1="5" x2="5" y2="19"></line></svg>';

    function prefersReducedMotion() {
        return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    }

    function ensureRoot() {
        if (root) return;

        root = document.createElement('div');
        root.className = 'trm';
        root.id = 'trm-root';
        root.setAttribute('aria-hidden', 'true');
        root.innerHTML =
            '<div class="trm-overlay" data-trm-dismiss></div>' +
            '<div class="trm-panel" role="dialog" aria-modal="true" aria-labelledby="trm-title" tabindex="-1">' +
                '<span class="trm-corner trm-corner-tl"></span>' +
                '<span class="trm-corner trm-corner-tr"></span>' +
                '<span class="trm-corner trm-corner-bl"></span>' +
                '<span class="trm-corner trm-corner-br"></span>' +
                '<button class="trm-close" type="button" aria-label="Close" data-trm-dismiss>' + CLOSE_ICON + '</button>' +
                '<header class="trm-head">' +
                    '<div class="trm-head-text">' +
                        '<p class="trm-eyebrow" id="trm-eyebrow"></p>' +
                        '<h2 class="trm-title" id="trm-title"></h2>' +
                    '</div>' +
                '</header>' +
                '<div class="trm-body" id="trm-body"></div>' +
            '</div>';

        document.body.appendChild(root);

        panel = root.querySelector('.trm-panel');
        bodyEl = root.querySelector('.trm-body');
        titleEl = root.querySelector('.trm-title');
        eyebrowEl = root.querySelector('.trm-eyebrow');
        headEl = root.querySelector('.trm-head');

        // Dismiss on backdrop / close-button (anything tagged data-trm-dismiss).
        root.addEventListener('click', function (event) {
            if (event.target.closest('[data-trm-dismiss]')) {
                close();
            }
        });

        document.addEventListener('keydown', function (event) {
            if (event.key === 'Escape' && isOpen()) {
                close();
            }
        });
    }

    function lockScroll(lock) {
        // Set overflow directly rather than via freezerScrolling(), which restores
        // `overflow: scroll` and would force a permanent (empty) horizontal scrollbar.
        document.body.style.overflow = lock ? 'hidden' : '';
    }

    function open(opts) {
        opts = opts || {};
        ensureRoot();

        var eyebrow = opts.eyebrow || '';
        var title = opts.title || '';
        var hasHeader = !!(eyebrow || title);

        onCloseCb = typeof opts.onClose === 'function' ? opts.onClose : null;

        eyebrowEl.textContent = eyebrow;
        eyebrowEl.style.display = eyebrow ? '' : 'none';
        titleEl.textContent = title;
        headEl.style.display = hasHeader ? '' : 'none';
        panel.classList.toggle('trm-panel-bare', !hasHeader);

        bodyEl.innerHTML = opts.body || '';

        panel.classList.remove('trm-variant-strava');
        if (opts.variant) panel.classList.add('trm-variant-' + opts.variant);

        lastFocus = document.activeElement;
        root.classList.remove('trm-closing');
        root.classList.add('trm-open');
        root.setAttribute('aria-hidden', 'false');
        lockScroll(true);

        // Restart the entrance animation if the panel was already mounted.
        panel.style.animation = 'none';
        // eslint-disable-next-line no-unused-expressions
        panel.offsetHeight;
        panel.style.animation = '';

        requestAnimationFrame(function () { panel.focus(); });
    }

    function setBody(html) {
        ensureRoot();
        bodyEl.innerHTML = html || '';
    }

    function finishClose() {
        root.classList.remove('trm-open', 'trm-closing');
        root.setAttribute('aria-hidden', 'true');
        lockScroll(false);

        var cb = onCloseCb;
        onCloseCb = null;

        if (lastFocus && typeof lastFocus.focus === 'function') {
            lastFocus.focus();
        }
        if (cb) cb();
    }

    function close() {
        if (!isOpen()) return;

        if (prefersReducedMotion()) {
            finishClose();
            return;
        }

        root.classList.add('trm-closing');
        var done = false;
        var settle = function () {
            if (done) return;
            done = true;
            finishClose();
        };
        panel.addEventListener('animationend', settle, { once: true });
        // Safety net if animationend never fires.
        setTimeout(settle, 320);
    }

    function isOpen() {
        return !!root && root.classList.contains('trm-open');
    }

    return { open: open, setBody: setBody, close: close, isOpen: isOpen };
})();

/* ── Backward-compatible globals ──────────────────────────────────────────── */
function closeModal() {
    TRModal.close();
}

function toggleModal(modalHTML) {
    if (modalHTML) {
        TRModal.open({ body: modalHTML });
    } else {
        TRModal.close();
    }
}
