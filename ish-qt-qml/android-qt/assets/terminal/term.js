window.onerror = function (message, source, lineno, colno, error) {
  var node = document.getElementById('jsError');
  if (node) node.textContent = '[JS ERROR] ' + String(message) + ' @ ' + String(source) + ':' + lineno + ':' + colno + '\r\n';
  return false;
};
function scriptLoadFailed(tag) {
  var node = document.getElementById('jsError');
  if (node) node.textContent = '[SCRIPT FAILED] ' + tag + '\r\n';
}
(function () {
  var output = document.getElementById('output');
  var terminal = document.getElementById('terminal');
  var cursor = document.getElementById('cursor');
  var socket = null;
  var fontSize = 14;
  var waitingForSession = true;
  var decoder = new TextDecoder('utf-8', { fatal: false });

  function append(value) {
    if (value == null) return;
    output.textContent += String(value);
    terminal.scrollTop = terminal.scrollHeight;
  }

  function markSessionConnected() {
    if (!waitingForSession) return;
    waitingForSession = false;
    output.textContent = 'iSH Qt terminal\r\n';
  }

  function send(value) {
    if (!value) return;
    if (socket && socket.readyState === WebSocket.OPEN)
      socket.send(JSON.stringify({ type: 'input', data: value }));
  }

  function connect(url) {
    if (!url || !/^wss?:/i.test(url)) return;
    try {
      socket = new WebSocket(url);
      socket.binaryType = 'arraybuffer';
      socket.onopen = function () { markSessionConnected(); append('\r\n'); terminal.focus(); };
      socket.onmessage = function (event) {
        markSessionConnected();
        if (event.data instanceof ArrayBuffer) {
          append(decoder.decode(new Uint8Array(event.data), { stream: true }));
        } else {
          try {
            var msg = JSON.parse(event.data);
            append(msg.data !== undefined ? msg.data :
                   msg.output !== undefined ? msg.output :
                   msg.text !== undefined ? msg.text : event.data);
          } catch (e) { append(event.data); }
        }
      };
      socket.onerror = function () { markSessionConnected(); append('\r\n[WebSocket error]\r\n'); };
      socket.onclose = function (event) {
        markSessionConnected();
        var code = event && typeof event.code === 'number' ? String(event.code) : '?';
        append('\r\n[session closed: code=' + code + ']\r\n');
      };
    } catch (error) { append('\r\n[connection error: ' + error + ']\r\n'); }
  }

  function controlCharacter(key) {
    if (!key || key.length !== 1) return '';
    var ch = key.toLowerCase();
    if (ch === ' ' || ch === '2') return '\x00';
    if (ch === '6') return '\x1e';
    if (ch === '-') return '\x1f';
    var code = ch.charCodeAt(0);
    if ((code >= 97 && code <= 122) || '@^[\\]_'.indexOf(ch) >= 0)
      return String.fromCharCode(code & 0x1f);
    return '';
  }

  terminal.addEventListener('keydown', function (event) {
    var value = '';
    if (event.key === 'Enter') value = '\r';
    else if (event.key === 'Backspace') value = '\x7f';
    else if (event.key === 'Tab') value = '\t';
    else if (event.key === 'Escape') value = '\x1b';
    else if (event.key === 'ArrowUp') value = '\x1b[A';
    else if (event.key === 'ArrowDown') value = '\x1b[B';
    else if (event.key === 'ArrowRight') value = '\x1b[C';
    else if (event.key === 'ArrowLeft') value = '\x1b[D';
    else if (event.ctrlKey && event.key.length === 1) value = controlCharacter(event.key);
    else if (event.altKey && event.key.length === 1) value = '\x1b' + event.key;
    else if (event.key.length === 1 && !event.metaKey) value = event.key;
    if (value) {
      send(value);
      event.preventDefault();
      event.stopPropagation();
    }
  });

  window.ishSendInput = send;
  window.ishPaste = function () {
    try {
      if (navigator.clipboard && navigator.clipboard.readText)
        navigator.clipboard.readText().then(function (text) { send(text); }).catch(function () {});
    } catch (e) { }
  };
  window.ishFontStep = function (delta) {
    fontSize = Math.max(8, Math.min(36, fontSize + Number(delta || 0)));
    terminal.style.fontSize = fontSize + 'px';
  };
  window.ishFontReset = function () { fontSize = 14; terminal.style.fontSize = '14px'; };
  window.ishClear = function () { output.textContent = ''; };
  window.ishCopy = function () {
    var s = window.getSelection();
    try {
      if (s && navigator.clipboard && navigator.clipboard.writeText)
        navigator.clipboard.writeText(s.toString()).catch(function () {});
    } catch (e) { }
  };
  window.ishConnect = connect;
  window.ishWsDiagnostic = function () {
    if (!socket) return 'none';
    var names = ['CONNECTING', 'OPEN', 'CLOSING', 'CLOSED'];
    return (socket.readyState >= 0 && socket.readyState < names.length) ?
           names[socket.readyState] : String(socket.readyState);
  };
  window.ishReconnect = function () {
    attempts = 0;
    retry();
  };

  var attempts = 0;
  var timer = null;
  var retry = function () {
    if (socket && socket.readyState === WebSocket.OPEN) {
      clearInterval(timer);
      return;
    }
    attempts += 1;
    connect(wsParam);
    if (attempts >= 120) {
      clearInterval(timer);
      append('\r\n[gave up after ' + attempts + ' connection attempts]\r\n');
    } else if (attempts % 5 === 1) {
      append('\r\n[retrying connection attempt ' + attempts + ']\r\n');
    }
  };

  append('iSH Qt terminal\r\n');
  terminal.focus();
  var query = new URLSearchParams(location.search);
  var wsParam = query.get('ws');
  if (wsParam) {
    retry();
    timer = setInterval(retry, 2000);
  } else {
    append('Waiting for the native session…\r\n');
    append('[no ws= parameter: loaded without session URL]\r\n');
  }
})();
