// SageOx Session Viewer
// Provides interactive features for HTML session exports

(function() {
    'use strict';

    // state
    var currentMessageIndex = -1;
    var messages = [];
    var searchVisible = false;

    document.addEventListener('DOMContentLoaded', init);

    function init() {
        messages = Array.from(document.querySelectorAll('.message'));

        setupKeyboardNav();
        setupExpandCollapse();
        setupCopyButtons();
        formatTimestamps();
        setupSearch();
        setupTruncation();
        injectControlsUI();
        handleHashNavigation();
    }

    // inject controls toolbar
    function injectControlsUI() {
        var header = document.querySelector('.page-header');
        if (!header) return;

        var controls = document.createElement('div');
        controls.className = 'viewer-controls';
        controls.innerHTML = [
            '<button type="button" class="btn btn-expand-all" title="Expand all tool details (e)">',
            '    <span class="btn-icon">+</span> Expand All',
            '</button>',
            '<button type="button" class="btn btn-collapse-all" title="Collapse all tool details (c)">',
            '    <span class="btn-icon">-</span> Collapse All',
            '</button>',
            '<button type="button" class="btn btn-search" title="Search messages (/)">',
            '    <span class="btn-icon">?</span> Search',
            '</button>',
            '<span class="nav-hint">j/k: navigate | e/c: expand/collapse | /: search</span>'
        ].join('\n');

        header.insertAdjacentElement('afterend', controls);

        controls.querySelector('.btn-expand-all').addEventListener('click', expandAll);
        controls.querySelector('.btn-collapse-all').addEventListener('click', collapseAll);
        controls.querySelector('.btn-search').addEventListener('click', toggleSearch);
    }

    // keyboard navigation
    function setupKeyboardNav() {
        document.addEventListener('keydown', function(e) {
            // ignore when typing in input fields
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                if (e.key === 'Escape') {
                    e.target.blur();
                    hideSearch();
                }
                return;
            }

            switch (e.key) {
                case 'j':
                    navigateMessage(1);
                    break;
                case 'k':
                    navigateMessage(-1);
                    break;
                case '/':
                    e.preventDefault();
                    toggleSearch();
                    break;
                case 'Escape':
                    hideSearch();
                    clearHighlight();
                    break;
                case 'e':
                    expandAll();
                    break;
                case 'c':
                    collapseAll();
                    break;
                case 'g':
                    // gg to go to top (simplified: single g goes to top)
                    navigateToMessage(0);
                    break;
                case 'G':
                    // G to go to bottom
                    navigateToMessage(messages.length - 1);
                    break;
            }
        });
    }

    function navigateMessage(direction) {
        if (messages.length === 0) return;

        var newIndex = currentMessageIndex + direction;
        if (newIndex >= 0 && newIndex < messages.length) {
            navigateToMessage(newIndex);
        }
    }

    function navigateToMessage(index) {
        if (index < 0 || index >= messages.length) return;

        // remove current highlight
        if (currentMessageIndex >= 0 && messages[currentMessageIndex]) {
            messages[currentMessageIndex].classList.remove('message-focused');
        }

        currentMessageIndex = index;
        var message = messages[currentMessageIndex];

        message.classList.add('message-focused');
        message.scrollIntoView({ behavior: 'smooth', block: 'center' });

        // update URL hash for direct linking
        if (message.id) {
            history.replaceState(null, '', '#' + message.id);
        }
    }

    function clearHighlight() {
        messages.forEach(function(msg) {
            msg.classList.remove('message-focused');
        });
        currentMessageIndex = -1;
    }

    // expand/collapse tool details
    function setupExpandCollapse() {
        // legacy support: tool-call-header click toggles
        document.querySelectorAll('.tool-call-header').forEach(function(header) {
            header.addEventListener('click', function() {
                var content = this.nextElementSibling;
                if (content && content.classList.contains('tool-call-content')) {
                    content.classList.toggle('collapsed');
                    this.classList.toggle('expanded');
                }
            });
        });

        // delegate click events on details elements
        document.addEventListener('click', function(e) {
            var toggle = e.target.closest('.tool-toggle');
            if (toggle) {
                var details = toggle.closest('.tool-details');
                if (details) {
                    details.classList.toggle('expanded');
                }
            }
        });

        // handle native details/summary elements
        document.querySelectorAll('details.tool-details').forEach(function(details) {
            details.addEventListener('toggle', function() {
                if (details.open) {
                    details.classList.add('expanded');
                } else {
                    details.classList.remove('expanded');
                }
            });
        });
    }

    function expandAll() {
        // native details elements
        document.querySelectorAll('details.tool-details').forEach(function(el) {
            el.open = true;
            el.classList.add('expanded');
        });

        // custom tool-details divs
        document.querySelectorAll('.tool-details').forEach(function(el) {
            el.classList.add('expanded');
        });

        // legacy tool-call-content
        document.querySelectorAll('.tool-call-content').forEach(function(el) {
            el.classList.remove('collapsed');
        });
        document.querySelectorAll('.tool-call-header').forEach(function(el) {
            el.classList.add('expanded');
        });

        // truncated content
        document.querySelectorAll('.truncated-content').forEach(function(el) {
            el.classList.add('expanded');
        });
        document.querySelectorAll('.btn-truncate-toggle').forEach(function(btn) {
            btn.textContent = 'Show less';
        });
    }

    function collapseAll() {
        // native details elements
        document.querySelectorAll('details.tool-details').forEach(function(el) {
            el.open = false;
            el.classList.remove('expanded');
        });

        // custom tool-details divs
        document.querySelectorAll('.tool-details').forEach(function(el) {
            el.classList.remove('expanded');
        });

        // legacy tool-call-content
        document.querySelectorAll('.tool-call-content').forEach(function(el) {
            el.classList.add('collapsed');
        });
        document.querySelectorAll('.tool-call-header').forEach(function(el) {
            el.classList.remove('expanded');
        });

        // truncated content
        document.querySelectorAll('.truncated-content').forEach(function(el) {
            el.classList.remove('expanded');
        });
        document.querySelectorAll('.btn-truncate-toggle').forEach(function(btn) {
            btn.textContent = 'Show more';
        });
    }

    // copy to clipboard
    function setupCopyButtons() {
        // legacy: code-block copy buttons
        document.querySelectorAll('.code-block').forEach(function(block) {
            if (block.querySelector('.copy-button')) return; // already has button

            var button = document.createElement('button');
            button.className = 'copy-button';
            button.textContent = 'Copy';
            button.addEventListener('click', function() {
                var code = block.querySelector('code');
                if (code) {
                    copyToClipboard(code.textContent, button);
                }
            });
            block.appendChild(button);
        });

        // delegate for btn-copy buttons
        document.addEventListener('click', function(e) {
            var copyBtn = e.target.closest('.btn-copy');
            if (!copyBtn) return;

            var targetSelector = copyBtn.dataset.target;
            var content = '';

            if (targetSelector) {
                var target = document.querySelector(targetSelector);
                if (target) {
                    content = target.textContent || target.innerText;
                }
            } else {
                // fallback: find nearest code/pre element
                var container = copyBtn.closest('.tool-output, .tool-input, .message-content');
                if (container) {
                    var codeEl = container.querySelector('pre, code');
                    content = codeEl ? codeEl.textContent : container.textContent;
                }
            }

            if (content) {
                copyToClipboard(content, copyBtn);
            }
        });
    }

    function copyToClipboard(text, button) {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(function() {
                showCopyFeedback(button, true);
            }).catch(function() {
                fallbackCopy(text, button);
            });
        } else {
            fallbackCopy(text, button);
        }
    }

    function fallbackCopy(text, button) {
        var textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();

        try {
            document.execCommand('copy');
            showCopyFeedback(button, true);
        } catch (err) {
            showCopyFeedback(button, false);
        }

        document.body.removeChild(textarea);
    }

    function showCopyFeedback(button, success) {
        var originalText = button.textContent;
        button.textContent = success ? 'Copied!' : 'Failed';
        button.classList.add(success ? 'copy-success' : 'copy-error');

        setTimeout(function() {
            button.textContent = originalText;
            button.classList.remove('copy-success', 'copy-error');
        }, 1500);
    }

    // timestamp formatting - convert UTC to local time
    function formatTimestamps() {
        document.querySelectorAll('[data-timestamp]').forEach(function(el) {
            var utcString = el.dataset.timestamp;
            if (!utcString) return;

            try {
                var date = new Date(utcString);
                if (isNaN(date.getTime())) return;

                var options = {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                    hour12: true
                };

                var localTime = date.toLocaleString(undefined, options);

                el.dataset.utc = utcString;
                el.textContent = localTime;
                el.title = 'UTC: ' + utcString;
            } catch (err) {
                // keep original on error
            }
        });
    }

    // search functionality
    function setupSearch() {
        var searchOverlay = document.createElement('div');
        searchOverlay.className = 'search-overlay';
        searchOverlay.innerHTML = [
            '<div class="search-container">',
            '    <input type="text" class="search-input" placeholder="Search messages..." autocomplete="off">',
            '    <div class="search-results">',
            '        <span class="search-count"></span>',
            '        <button type="button" class="btn-search-prev" title="Previous (Shift+Enter)">&uarr;</button>',
            '        <button type="button" class="btn-search-next" title="Next (Enter)">&darr;</button>',
            '    </div>',
            '    <button type="button" class="btn-search-close" title="Close (Esc)">&times;</button>',
            '</div>'
        ].join('\n');
        document.body.appendChild(searchOverlay);

        var input = searchOverlay.querySelector('.search-input');
        var countEl = searchOverlay.querySelector('.search-count');
        var prevBtn = searchOverlay.querySelector('.btn-search-prev');
        var nextBtn = searchOverlay.querySelector('.btn-search-next');
        var closeBtn = searchOverlay.querySelector('.btn-search-close');

        var searchResults = [];
        var searchIndex = -1;

        input.addEventListener('input', debounce(function() {
            var query = input.value.trim().toLowerCase();
            searchResults = [];
            searchIndex = -1;

            // clear previous highlights
            document.querySelectorAll('.search-highlight').forEach(function(el) {
                el.classList.remove('search-highlight');
            });

            if (query.length < 2) {
                countEl.textContent = '';
                return;
            }

            // search through messages
            messages.forEach(function(msg, idx) {
                var text = msg.textContent.toLowerCase();
                if (text.indexOf(query) !== -1) {
                    searchResults.push(idx);
                    msg.classList.add('search-highlight');
                }
            });

            if (searchResults.length > 0) {
                countEl.textContent = searchResults.length + ' found';
                searchIndex = 0;
                navigateToMessage(searchResults[0]);
            } else {
                countEl.textContent = 'No results';
            }
        }, 200));

        input.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                if (e.shiftKey) {
                    searchPrev();
                } else {
                    searchNext();
                }
            }
        });

        prevBtn.addEventListener('click', searchPrev);
        nextBtn.addEventListener('click', searchNext);
        closeBtn.addEventListener('click', hideSearch);

        function searchNext() {
            if (searchResults.length === 0) return;
            searchIndex = (searchIndex + 1) % searchResults.length;
            navigateToMessage(searchResults[searchIndex]);
            countEl.textContent = (searchIndex + 1) + '/' + searchResults.length;
        }

        function searchPrev() {
            if (searchResults.length === 0) return;
            searchIndex = (searchIndex - 1 + searchResults.length) % searchResults.length;
            navigateToMessage(searchResults[searchIndex]);
            countEl.textContent = (searchIndex + 1) + '/' + searchResults.length;
        }
    }

    function toggleSearch() {
        var overlay = document.querySelector('.search-overlay');
        if (!overlay) return;

        searchVisible = !searchVisible;
        if (searchVisible) {
            overlay.classList.add('visible');
        } else {
            overlay.classList.remove('visible');
        }

        if (searchVisible) {
            var input = overlay.querySelector('.search-input');
            input.focus();
            input.select();
        }
    }

    function hideSearch() {
        var overlay = document.querySelector('.search-overlay');
        if (overlay) {
            overlay.classList.remove('visible');
            searchVisible = false;
        }

        // clear search highlights
        document.querySelectorAll('.search-highlight').forEach(function(el) {
            el.classList.remove('search-highlight');
        });
    }

    // long content truncation
    function setupTruncation() {
        var MAX_LINES = 50;
        var LINE_HEIGHT = 20;
        var MAX_HEIGHT = MAX_LINES * LINE_HEIGHT;

        document.querySelectorAll('.tool-output pre, .tool-input pre').forEach(function(pre) {
            if (pre.scrollHeight > MAX_HEIGHT) {
                var wrapper = document.createElement('div');
                wrapper.className = 'truncated-content';

                pre.parentNode.insertBefore(wrapper, pre);
                wrapper.appendChild(pre);

                var toggle = document.createElement('button');
                toggle.type = 'button';
                toggle.className = 'btn-truncate-toggle';
                toggle.textContent = 'Show more';
                wrapper.appendChild(toggle);

                toggle.addEventListener('click', function() {
                    var isExpanded = wrapper.classList.toggle('expanded');
                    toggle.textContent = isExpanded ? 'Show less' : 'Show more';
                });
            }
        });
    }

    // utility: debounce function calls
    function debounce(fn, delay) {
        var timeout;
        return function() {
            var context = this;
            var args = arguments;
            clearTimeout(timeout);
            timeout = setTimeout(function() {
                fn.apply(context, args);
            }, delay);
        };
    }

    // handle direct links to messages via URL hash
    function handleHashNavigation() {
        var hash = window.location.hash;
        if (hash && hash.indexOf('#msg-') === 0) {
            var target = document.querySelector(hash);
            if (target) {
                var index = messages.indexOf(target);
                if (index >= 0) {
                    setTimeout(function() {
                        navigateToMessage(index);
                    }, 100);
                }
            }
        }
    }

    // listen for hash changes for navigation
    window.addEventListener('hashchange', handleHashNavigation);

    // iframe height communication
    if (window !== window.parent) {
        function postHeight() {
            window.parent.postMessage(
                { type: 'sageox-session-resize', height: document.documentElement.scrollHeight },
                '*'
            );
        }
        window.addEventListener('load', postHeight);
        window.addEventListener('resize', postHeight);
        new MutationObserver(postHeight).observe(document.body, { childList: true, subtree: true, attributes: true });
    }

})();
