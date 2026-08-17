(() => {
  const output = document.getElementById('output');
  const terminal = document.getElementById('terminal');
  const cursor = document.getElementById('cursor');
  let socket = null;
  let fontSize = 14;
  let waitingForSession = true;
  const decoder = new TextDecoder('utf-8', { fatal: false });

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
      socket.onopen = () => { markSessionConnected(); append('\r\n'); terminal.focus(); };
      socket.onmessage = event => {
        markSessionConnected();
        if (event.data instanceof ArrayBuffer)
          append(decoder.decode(new Uint8Array(event.data), { stream: true }));
        else {
          try {
            const msg = JSON.parse(event.data);
            append(msg.data ?? msg.output ?? msg.text ?? event.data);
          } catch (_) { append(event.data); }
        }
      };
      socket.onerror = () => { markSessionConnected(); append('\r\n[WebSocket error]\r\n'); };
      socket.onclose = event => {
        markSessionConnected();
        const code = event != null && typeof event.code === 'number' ? String(event.code) : '?';
        append('\r\n[session closed: code=' + code + ']\r\n');
      };
    } catch (error) { append('\r\n[connection error: ' + error + ']\r\n'); }
  }

  function controlCharacter(key) {
    if (!key || key.length !== 1) return '';
    const ch = key.toLowerCase();
    if (ch === ' ' || ch === '2') return '\x00';
    if (ch === '6') return '\x1e';
    if (ch === '-') return '\x1f';
    const code = ch.charCodeAt(0);
    if ((code >= 97 && code <= 122) || '@^[\\]_'.includes(ch))
      return String.fromCharCode(code & 0x1f);
    return '';
  }

  terminal.addEventListener('keydown', event => {
    let value = '';
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
  window.ishPaste = async () => { try { send(await navigator.clipboard.readText()); } catch (_) {} };
  window.ishFontStep = delta => { fontSize = Math.max(8, Math.min(36, fontSize + Number(delta || 0))); terminal.style.fontSize = fontSize + 'px'; };
  window.ishFontReset = () => { fontSize = 14; terminal.style.fontSize = '14px'; };
  window.ishClear = () => { output.textContent = ''; };
  window.ishCopy = () => { const s = window.getSelection(); if (s) navigator.clipboard?.writeText(s.toString()); };
  window.ishConnect = connect;
  window.ishWsDiagnostic = () => {
    if (!socket) return "none";
    const names = ["CONNECTING", "OPEN", "CLOSING", "CLOSED"];
    return names[socket.readyState] ?? String(socket.readyState);
  };

  append('iSH Qt terminal\r\n');
  terminal.focus();
  const query = new URLSearchParams(location.search);
  const wsParam = query.get('ws');
  if (wsParam) {
    let attempts = 0;
    let timer = null;
    const retry = () => {
      if (socket && socket.readyState === WebSocket.OPEN) {
        clearInterval(timer);
        return;
      }
      attempts += 1;
      connect(wsParam);
      if (attempts >= 30) clearInterval(timer);
    };
    retry();
    timer = setInterval(retry, 2000);
    window.ishReconnect = () => { attempts = 0; retry(); };
  } else {
    append('Waiting for the native session…\r\n');
    append('[no ws= parameter: loaded without session URL]\r\n');
  }
})();
