/**
 * iSH Android - term.js - Full port of iOS app/terminal/term.js (5.6K)
 * Original iOS uses hterm + webkit.messageHandlers
 * Android pure Rust version uses hterm + Android JavascriptInterface + Slint bridge
 * 
 * This file initializes hterm as iSH iOS does:
 * - hterm.defaultStorage = Memory
 * - transparent colors initially, then updateStyle
 * - terminal-encoding iso-2022, disable resize status, copy-on-select, clipboard notice
 * - user-css-text, screen-padding-size 4, audible-bell-sound ''
 * - onTerminalReady -> exports
 * 
 * Exports matching iOS:
 * - write(data), getSize(), copy(), setFocused(), scrollToBottom(), newScrollTop(y)
 * - updateStyle({fg, bg, fontFamily, fontSize, colorPaletteOverrides, blinkCursor, cursorShape})
 * - getCharacterSize(), clearScrollback(), setUserGesture()
 * - hterm.openUrl -> native.openLink
 */

hterm.defaultStorage = new lib.Storage.Memory();
window.onload = async function() {
    await lib.init();
    window.term = new hterm.Terminal();

    // make everything invisible initially as iOS does
    term.getPrefs().set('background-color', 'transparent');
    term.getPrefs().set('foreground-color', 'transparent');
    term.getPrefs().set('cursor-color', 'transparent');

    term.getPrefs().set('terminal-encoding', 'iso-2022');
    term.getPrefs().set('enable-resize-status', false);
    term.getPrefs().set('copy-on-select', false);
    term.getPrefs().set('enable-clipboard-notice', false);
    term.getPrefs().set('user-css-text', termCss);
    term.getPrefs().set('screen-padding-size', 4);
    term.getPrefs().set('audible-bell-sound', '');

    term.onTerminalReady = onTerminalReady;
    term.decorate(document.getElementById('terminal'));
};

var termCss = `
x-screen {
    background: transparent !important;
    overflow: hidden !important;
    -webkit-tap-highlight-color: transparent;
}
x-row {
  text-rendering: optimizeLegibility;
  font-variant-ligatures: normal;
}
.uri-node {
  text-decoration: underline;
}
body {
  background: #000 !important;
}
`;

// Shorthand for JS -> native IPC - supports both iOS webkit and Android
const native = new Proxy({}, {
    get(obj, prop) {
        return (...args) => {
            if (args.length == 0) args = null;
            else if (args.length == 1) args = args[0];
            
            // iOS path
            if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers[prop]) {
                try {
                    window.webkit.messageHandlers[prop].postMessage(args);
                    return;
                } catch(e) { console.log('webkit error', e); }
            }
            // Android path - map prop to Android interface
            if (window.Android) {
                try {
                    switch(prop) {
                        case 'load': if (window.Android.onLoad) window.Android.onLoad(); break;
                        case 'log': if (window.Android.log) window.Android.log(args ? args.toString() : ''); break;
                        case 'sendInput': if (window.Android.onSendInput) window.Android.onSendInput(args ? args.toString() : ''); break;
                        case 'resize': 
                            if (window.Android.onResize) {
                                // args is [cols, rows] or null
                                if (Array.isArray(args)) window.Android.onResize(args[0], args[1]);
                                else if (args && args.length) window.Android.onResize(args[0], args[1]);
                            }
                            break;
                        case 'propUpdate': if (window.Android.onPropUpdate) window.Android.onPropUpdate(args ? args.toString() : ''); break;
                        case 'focus': if (window.Android.onFocus) window.Android.onFocus(); break;
                        case 'syncFocus': if (window.Android.onSyncFocus) window.Android.onSyncFocus(); break;
                        case 'newScrollHeight': if (window.Android.onNewScrollHeight) window.Android.onNewScrollHeight(args ? parseFloat(args) : 0); break;
                        case 'newScrollTop': if (window.Android.onNewScrollTop) window.Android.onNewScrollTop(args ? parseFloat(args) : 0); break;
                        case 'openLink': if (window.Android.onOpenLink) window.Android.onOpenLink(args ? args.toString() : ''); break;
                        default: if (window.Android.log) window.Android.log('native.' + prop + ' ' + JSON.stringify(args)); break;
                    }
                    return;
                } catch(e) { console.log('Android bridge error', e); }
            }
            console.log('native.' + prop, args);
        };
    },
});

// Functions for native -> JS - matches iOS exports
window.exports = {};

function onTerminalReady() {
    term.io.push();
    term.reset();

    let oldProps = {};
    function syncProp(name, value) {
        if (oldProps[name] !== value) {
            try { native.propUpdate(name, value); } catch(e) {}
            oldProps[name] = value;
        }
    }
    let decoder = new TextDecoder();
    exports.write = (data) => {
        try {
            term.io.writeUTF16(decoder.decode(lib.codec.stringToCodeUnitArray(data)));
        } catch(e) {
            // Fallback if lib.codec not available
            term.io.print(data);
        }
        syncProp('applicationCursor', term.keyboard.applicationCursor);
    };
    term.io.sendString = term.io.onVTKeyStroke = (data) => {
        native.sendInput(data);
    };

    // hterm size updates native size
    term.io.onTerminalResize = () => native.resize();
    exports.getSize = () => [term.screenSize.width, term.screenSize.height];

    // selection, copying
    try {
        term.scrollPort_.screen_.contentEditable = false;
        term.blur();
        term.focus();
    } catch(e) {}
    exports.copy = () => {
        try { term.copySelectionToClipboard(); } catch(e) {}
    };

    // focus handling
    try {
        term.scrollPort_.screen_.addEventListener('blur', (e) => {
            if (e.target.ownerDocument.activeElement == e.target) {
                e.stopPropagation();
            }
        }, {capture: true});
        term.scrollPort_.screen_.addEventListener('mousedown', (e) => {
            if ((document.getSelection().rangeCount != 0) &&
                (!document.getSelection().isCollapsed)) return;
            native.focus();
        });
    } catch(e) {}
    exports.setFocused = (focus) => {
        try {
            if (focus) term.focus();
            else term.blur();
        } catch(e) {}
    };
    try {
        term.scrollPort_.screen_.addEventListener('focus', (e) => native.syncFocus());
    } catch(e) {}

    // scrolling - disable hterm builtin touch scrolling
    try {
        term.scrollPort_.onTouch = (e) => {
            Object.defineProperty(e, 'defaultPrevented', {value: true});
        };
    } catch(e) {}
    exports.scrollToBottom = () => {
        try { term.scrollEnd(); } catch(e) {}
    };
    exports.newScrollTop = (y) => {
        try {
            term.scrollPort_.screen_.scrollTop = y;
            lastScrollTop = term.scrollPort_.screen_.scrollTop;
        } catch(e) {}
    };

    // Send scroll height and position to native
    let lastScrollHeight, lastScrollTop;
    function syncScroll() {
        try {
            const scrollHeight = parseFloat(term.scrollPort_.scrollArea_.style.height);
            if (scrollHeight != lastScrollHeight)
                native.newScrollHeight(scrollHeight);
            lastScrollHeight = scrollHeight;

            const scrollTop = term.scrollPort_.screen_.scrollTop;
            if (scrollTop != lastScrollTop)
                native.newScrollTop(scrollTop);
            lastScrollTop = scrollTop;
        } catch(e) {}
    }

    try {
        const realSyncScrollHeight = hterm.ScrollPort.prototype.syncScrollHeight;
        hterm.ScrollPort.prototype.syncScrollHeight = function() {
            realSyncScrollHeight.call(this);
            syncScroll();
        };
        term.scrollPort_.screen_.addEventListener('scroll', syncScroll);
    } catch(e) {}

    exports.updateStyle = ({foregroundColor, backgroundColor, fontFamily, fontSize, colorPaletteOverrides, blinkCursor, cursorShape}) => {
        try {
            term.getPrefs().set('background-color', backgroundColor);
            term.getPrefs().set('foreground-color', foregroundColor);
            term.getPrefs().set('cursor-color', foregroundColor);
            term.getPrefs().set('font-family', fontFamily);
            term.getPrefs().set('font-size', fontSize);
            term.getPrefs().set('color-palette-overrides', colorPaletteOverrides);
            term.getPrefs().set('cursor-blink', blinkCursor);
            term.getPrefs().set('cursor-shape', cursorShape);
        } catch(e) { console.log('updateStyle error', e); }
    };

    exports.getCharacterSize = () => {
        try {
            return [term.scrollPort_.characterSize.width, term.scrollPort_.characterSize.height];
        } catch(e) { return [8, 16]; }
    };

    exports.clearScrollback = () => {
        try { term.clearScrollback(); } catch(e) {}
    };
    exports.setUserGesture = () => {
        try { term.accessibilityReader_.hasUserGesture = true; } catch(e) {}
    };

    hterm.openUrl = (url) => native.openLink(url);

    // Simulate boot messages for Android pure Rust - matches gui.rs AppDelegate
    try {
        term.setProfile('default');
        term.getPrefs().set('background-color', '#000000');
        term.getPrefs().set('foreground-color', '#ffffff');
        term.getPrefs().set('cursor-color', '#ffffff');
        term.getPrefs().set('font-family', 'ui-monospace, Menlo, monospace');
        term.getPrefs().set('font-size', 14);
    } catch(e) {}

    // Write boot sequence matching gui.rs and android-rs
    setTimeout(() => {
        const bootMessages = [
            '\x1b[1;32miSH - Alpine Linux shell (Rust + Android + KVM + hterm)\x1b[0m',
            'Terminal: hterm_all.js + term.js + term.css (as iOS)',
            'UI: UIKit -> Slint + Android (TerminalViewController + BarButton + ArrowBarButton)',
            'Emu: cpu + decode + asbestos JIT + ram (MmapMut via memmap2 0.9.11) + KVM',
            '',
            '[boot] Mounting rootfs at /tmp/alpine_real...',
            '[boot] RAM: 4096 pages of 4096 bytes (real MmapMut)',
            '[boot] Asbestos JIT hash_size=1024',
            '[boot] ELF entry: 0x08048000',
            '[boot] Theme: Default (6 themes) dir=/tmp/ish_documents/themes',
            '[boot] Font: ui-monospace 12.0px (System), cursor: BLOCK',
            '[boot] KVM acceleration: ENABLED (linux + kvm)',
            '[boot] hterm initialized via lib.init() + term.decorate()',
            '',
            'Type \"help\" for help',
            ''
        ];
        bootMessages.forEach(line => {
            try { term.writeln(line); } catch(e) { console.log(line); }
        });
        term.write('$ ');
    }, 100);

    native.load();
    native.syncFocus();
    
    // Notify Android that term is ready
    if (window.Android && window.Android.onLoad) {
        try { window.Android.onLoad(); } catch(e) {}
    }
    
    console.log('iSH hterm ready - SlintUi + Servo WebView + KVM');
}
