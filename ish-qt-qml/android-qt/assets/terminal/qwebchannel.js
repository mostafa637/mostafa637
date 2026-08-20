(function(global) {
  'use strict';
  if (global.QWebChannel) return;
  function QWebChannel(transport, initCallback) {
    this.transport = transport;
    this.objects = {};
    var self = this;
    transport.onmessage = function(message) {
      var data = typeof message.data === 'string' ? JSON.parse(message.data) : message.data;
      if (data && data.type === 'init' && data.objects) {
        self.objects = data.objects;
        if (initCallback) initCallback(self.objects);
      }
    };
    transport.send = transport.send || function(payload) {
      if (transport.websocket) transport.websocket.send(JSON.stringify(payload));
    };
  }
  global.QWebChannel = QWebChannel;
})(window);
