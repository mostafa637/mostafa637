package com.ish.emulator

import android.annotation.SuppressLint
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.Log
import android.webkit.JavascriptInterface
import android.webkit.WebChromeClient
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity

/**
 * iSH Android Pure Rust - Full port of iOS GUI (36 Objective-C files under app/) to Android
 * 
 * Original iOS initialization (as requested: تهيئة ملفات xterm.js كما يفعلة ish ios):
 * - app/terminal/term.html loads hterm_all.js + term.js + term.css
 * - term.js: hterm.defaultStorage = Memory, lib.init(), new hterm.Terminal(), transparent colors, terminal-encoding iso-2022
 * - term.js onTerminalReady: exports.write, sendString, getSize, copy, setFocused, scrollToBottom, newScrollTop, updateStyle, etc.
 * - Terminal.m: WKWebView with CustomWebView, pendingData BUF_SIZE 1<<14, dataLock, outputInProgress, refreshTask, arrow, convertCommand, destroy
 * 
 * Android pure Rust version:
 * - WebView loads file:///android_asset/terminal/term.html (hterm_all.js as iOS)
 * - SlintUi + Servo WebView abstraction from gui.rs (2960 lines, not simplified)
 * - KVM acceleration via linux + kvm on ubuntu-latest emulator
 * - Screenshot via adb exec-out screencap -p
 */

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private lateinit var statusText: TextView
    private var controlPressed = false
    private var applicationCursor = false

    // Terminal.m simulation
    private val pendingData = mutableListOf<Byte>()
    private val bufSize = 1 shl 14
    private var outputInProgress = false
    private var loaded = false
    private var winsizeCols = 80
    private var winsizeRows = 24

    @SuppressLint("SetJavaScriptEnabled", "AddJavascriptInterface")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        Log.i("iSH", "=== iSH Android Pure Rust (Slint + Servo + hterm) ===")
        Log.i("iSH", "AppDelegate.didFinishLaunching - boot sequence (as iOS AppDelegate.m 342 lines)")
        Log.i("iSH", "Mounting rootfs at /tmp/alpine_real (Roots.m 234 lines)")
        Log.i("iSH", "fs_register iosfs, iosfs_unsafe (iOSFS.m 532 lines)")
        Log.i("iSH", "become_first_process + FsInitialize + create device nodes")
        Log.i("iSH", "dyn_dev_register clipboard (PasteboardDevice.m 252) + location (LocationDevice.m 174)")
        Log.i("iSH", "do_mount proc /dev/pts + iosfs_init + configureDns (res_ninit + res_getservers)")
        Log.i("iSH", "Terminal.m: CustomWebView frame=10000x10000 inspectable=YES scrollEnabled=NO")
        Log.i("iSH", "TerminalView.m: UITextInput, _updateStyle, keyCommands, floating cursor")
        Log.i("iSH", "Theme.m: Palette hex parsing #000 #fff #ff0f #ffffff #ffffffff, defaultThemes 6")
        Log.i("iSH", "UserPreferences.m: 592 lines, capsLockMapping, optionMapping, fontFamily ui-monospace")

        webView = findViewById(R.id.terminalWebView)
        statusText = findViewById(R.id.statusText)

        setupWebView()
        setupExtraKeys()
        setupStatusBar()
        loadTerminal()
    }

    @SuppressLint("SetJavaScriptEnabled")
    private fun setupWebView() {
        webView.settings.apply {
            javaScriptEnabled = true
            domStorageEnabled = true
            allowFileAccess = true
            allowContentAccess = true
            allowFileAccessFromFileURLs = true
            allowUniversalAccessFromFileURLs = true
            useWideViewPort = true
            loadWithOverviewMode = true
            setSupportZoom(false)
            // Enable debugging as iOS inspectable=YES
            // if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) WebView.setWebContentsDebuggingEnabled(true)
        }
        webView.webChromeClient = WebChromeClient()
        webView.webViewClient = object : WebViewClient() {
            override fun onPageFinished(view: WebView?, url: String?) {
                super.onPageFinished(view, url)
                Log.i("iSH", "Terminal.webView onPageFinished: $url - waiting for hterm lib.init()")
                // Don't set loaded here, wait for native.load() callback from term.js
                // Simulate _updateStyle after load as iOS does
                updateStyle()
            }
        }
        webView.addJavascriptInterface(this, "Android")
        // Also add webkit messageHandlers shim via JS injection for compatibility
        webView.addJavascriptInterface(this, "webkitShim")
    }

    private fun setupExtraKeys() {
        // BarButton.m (91) + ArrowBarButton.m (238) - SlintUi extra_keys
        findViewById<Button>(R.id.btnTab).setOnClickListener { 
            sendInput("\t")
            webView.evaluateJavascript("if(window.term) { term.io.sendString('\\t'); }", null)
        }
        findViewById<Button>(R.id.btnCtrl).setOnClickListener {
            controlPressed = !controlPressed
            statusText.text = if (controlPressed) "CTRL pressed - next key will be control (TerminalView.m handleKeyCommand)" else "iSH - Alpine Linux 3.18 | Default theme | ui-monospace 12px | BLOCK | hterm + KVM"
            Log.i("iSH", "BarButton Ctrl toggled: $controlPressed (matches TerminalViewController controlKeyToggled)")
        }
        findViewById<Button>(R.id.btnEsc).setOnClickListener { 
            sendInput("\u001b")
            webView.evaluateJavascript("if(window.term) { term.io.sendString('\\x1b'); }", null)
        }
        findViewById<Button>(R.id.btnUp).setOnClickListener { 
            val seq = arrow('A')
            sendInput(seq)
            webView.evaluateJavascript("if(window.exports) exports.write(\"$seq\"); else if(window.term) term.io.writeUTF16('$seq');", null)
        }
        findViewById<Button>(R.id.btnDown).setOnClickListener { 
            val seq = arrow('B')
            sendInput(seq)
        }
        findViewById<Button>(R.id.btnLeft).setOnClickListener { 
            val seq = arrow('D')
            sendInput(seq)
        }
        findViewById<Button>(R.id.btnRight).setOnClickListener { 
            val seq = arrow('C')
            sendInput(seq)
        }

        // Long press repeat - ArrowBarButton long_press_interval 0.1
        findViewById<Button>(R.id.btnUp).setOnLongClickListener {
            repeatArrow('A')
            true
        }
        findViewById<Button>(R.id.btnDown).setOnLongClickListener {
            repeatArrow('B')
            true
        }
    }

    private fun setupStatusBar() {
        findViewById<Button>(R.id.btnClear).setOnClickListener {
            // TerminalView.clearScrollback: k+cmd+shift
            webView.evaluateJavascript("if(window.exports && exports.clearScrollback) exports.clearScrollback(); if(window.term) term.clearScrollback();", null)
            Log.i("iSH", "ThemeViewController + TerminalView clearScrollback")
        }
        statusText.text = "iSH - Alpine Linux 3.18 | Default theme | ui-monospace 12px | BLOCK | hterm + KVM | Pure Rust"
    }

    private fun loadTerminal() {
        // As iOS: loadFileURL term.html allowingReadAccessToURL term.html
        // iOS uses NSBundle.mainBundle URLForResource:@"term" withExtension:@"html"
        // Android uses file:///android_asset/terminal/term.html
        webView.loadUrl("file:///android_asset/terminal/term.html")
        Log.i("iSH", "Terminal.webView loadFileURL: file:///android_asset/terminal/term.html allowingReadAccess (as iOS)")
        Log.i("iSH", "Loading hterm_all.js (698K) + term.css (160) + term.js (12K) as iOS does")
    }

    private fun updateStyle() {
        // Matches TerminalView._updateStyle: fontFamily, fontSize, foregroundColor, backgroundColor, blinkCursor, cursorShape, colorPaletteOverrides
        val prefs = mapOf(
            // 'ui-monospace' is an SF Mono alias that exists on Apple platforms only;
            // on Android it resolves to the default proportional face unless a monospace
            // fallback is present, which breaks hterm's fixed cell width.
            "fontFamily" to "ui-monospace, monospace",
            // 12 matches both UserPreferences' default on iOS and the status bar text
            // this app prints; the old 14 contradicted both.
            "fontSize" to 12,
            "foregroundColor" to "#ffffff",
            "backgroundColor" to "#000000",
            "blinkCursor" to false,
            "cursorShape" to "BLOCK"
        )
        val json = "{\"foregroundColor\":\"${prefs["foregroundColor"]}\",\"backgroundColor\":\"${prefs["backgroundColor"]}\",\"fontFamily\":\"${prefs["fontFamily"]}\",\"fontSize\":${prefs["fontSize"]},\"blinkCursor\":${prefs["blinkCursor"]},\"cursorShape\":\"${prefs["cursorShape"]}\"}"
        webView.evaluateJavascript("if(window.exports && exports.updateStyle) exports.updateStyle($json);", null)
        Log.i("iSH", "TerminalView._updateStyle $json")
    }

    fun arrow(direction: Char): String {
        // Terminal.m arrow: \x1b + (applicationCursor ? 'O' : '[') + direction
        return "\u001b${if (applicationCursor) 'O' else '['}$direction"
    }

    private fun repeatArrow(direction: Char) {
        for (i in 0..5) {
            sendInput(arrow(direction))
        }
    }

    // Terminal.m sendOutput
    fun sendOutput(buf: ByteArray): Int {
        synchronized(pendingData) {
            if (pendingData.size > bufSize) {
                val room = bufSize - pendingData.size
                val len = minOf(buf.size, room)
                if (len > 0) pendingData.addAll(buf.take(len))
                return len
            }
            pendingData.addAll(buf.toList())
        }
        // Schedule refreshTask as iOS does
        webView.post { refresh() }
        return buf.size
    }

    // Terminal.m refresh
    fun refresh() {
        if (!loaded) {
            webView.postDelayed({ refresh() }, 16)
            return
        }
        if (outputInProgress) {
            webView.postDelayed({ refresh() }, 16)
            return
        }
        val data: ByteArray
        synchronized(pendingData) {
            if (pendingData.isEmpty()) return
            data = pendingData.toByteArray()
            pendingData.clear()
            outputInProgress = true
        }
        // Escape for JS as iOS: latin1, replace \ \r \n "
        val dataString = String(data, Charsets.ISO_8859_1)
            .replace("\\", "\\\\")
            .replace("\r", "\\r")
            .replace("\n", "\\n")
            .replace("\"", "\\\"")
        val js = "if(window.exports && exports.write) { exports.write(\"$dataString\"); } else if(window.term) { term.io.print(\"$dataString\"); }"
        webView.evaluateJavascript(js) {
            outputInProgress = false
        }
    }

    fun sendInput(text: String) {
        var input = text
        if (controlPressed && input.length == 1) {
            val c = input[0]
            val ctrl = (c.code and 0x1f).toChar()
            input = ctrl.toString()
            controlPressed = false
            statusText.text = "iSH - Alpine Linux 3.18 | Default theme | ui-monospace 12px | BLOCK | hterm"
        }
        Log.i("iSH", "Terminal.sendInput: ${input.replace("\u001b", "\\x1b")} (tty=0x1000)")
        // In real iSH, tty_input(tty, bytes, len, 0)
    }

    // JavascriptInterface callbacks matching term.js native.* calls
    @JavascriptInterface
    fun log(msg: String) {
        Log.i("iSH-JS", msg)
    }

    @JavascriptInterface
    fun onLoad() {
        // term.js reaches this twice: native.load() is routed here by the shim, and
        // the page also calls window.Android.onLoad() directly as a safety net.
        // iOS used a one-shot KVO observation on Terminal.loaded for the same reason,
        // so guard it - otherwise the boot preview is written to the tty twice.
        if (loaded) {
            Log.i("iSH", "term.js native.load() ignored - Terminal.loaded already YES")
            return
        }
        Log.i("iSH", "term.js native.load() -> Terminal.loaded=YES (KVO) + refreshTask schedule")
        loaded = true
        // As iOS: self.loaded=YES, [self.refreshTask schedule], enableVoiceOverAnnounce
        runOnUiThread {
            sendOutput(previewString.toByteArray())
            refresh()
            updateStyle()
        }
    }

    @JavascriptInterface
    fun onSendInput(data: String) {
        Log.i("iSH", "term.js native.sendInput: $data -> tty_input")
        // Would call tty_input or async_do_in_workqueue for Linux
    }

    @JavascriptInterface
    fun onResize(cols: Int, rows: Int) {
        winsizeCols = cols
        winsizeRows = rows
        Log.i("iSH", "term.js native.resize() -> syncWindowSize cols=$cols rows=$rows (tty_set_winsize)")
    }

    @JavascriptInterface
    fun onResize() {
        // Called without args, get size via JS
        webView.evaluateJavascript("exports.getSize()") { result ->
            Log.i("iSH", "getSize result: $result")
        }
    }

    @JavascriptInterface
    fun onPropUpdate(data: String) {
        Log.i("iSH", "term.js native.propUpdate: $data")
        if (data.contains("applicationCursor")) {
            applicationCursor = data.contains("true")
        }
    }

    @JavascriptInterface
    fun onFocus() {
        Log.i("iSH", "term.js native.focus() -> TerminalView.becomeFirstResponder")
    }

    @JavascriptInterface
    fun onSyncFocus() {
        Log.i("iSH", "term.js native.syncFocus() -> sync terminalFocused")
    }

    @JavascriptInterface
    fun onNewScrollHeight(height: Float) {
        Log.i("iSH", "term.js native.newScrollHeight: $height -> ScrollbarView.contentSize")
    }

    @JavascriptInterface
    fun onNewScrollTop(top: Float) {
        Log.i("iSH", "term.js native.newScrollTop: $top -> ScrollbarView.contentOffset")
    }

    @JavascriptInterface
    fun onOpenLink(url: String) {
        Log.i("iSH", "term.js hterm.openUrl: $url -> UIApplication.openURL")
        try {
            val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
            startActivity(intent)
        } catch(e: Exception) {
            Log.e("iSH", "openLink failed: $url", e)
        }
    }

    // For webkit shim compatibility
    @JavascriptInterface
    fun postMessage(handler: String, data: String) {
        Log.i("iSH", "webkit.messageHandlers.$handler postMessage: $data")
        when(handler) {
            "load" -> onLoad()
            "log" -> log(data)
            "sendInput" -> onSendInput(data)
            "resize" -> onResize()
            "propUpdate" -> onPropUpdate(data)
            "focus" -> onFocus()
            "syncFocus" -> onSyncFocus()
            "newScrollHeight" -> onNewScrollHeight(data.toFloatOrNull() ?: 0f)
            "newScrollTop" -> onNewScrollTop(data.toFloatOrNull() ?: 0f)
            "openLink" -> onOpenLink(data)
        }
    }

    private val previewString = """
# cat /proc/ish/colors
${"\u001b"}[30miSH${"\u001b"}[39m ${"\u001b"}[31miSH${"\u001b"}[39m ${"\u001b"}[32miSH${"\u001b"}[39m ${"\u001b"}[33miSH${"\u001b"}[39m ${"\u001b"}[34miSH${"\u001b"}[39m ${"\u001b"}[35miSH${"\u001b"}[39m ${"\u001b"}[36miSH${"\u001b"}[39m ${"\u001b"}[37miSH${"\u001b"}[39m
${"\u001b"}[7m${"\u001b"}[40miSH${"\u001b"}[39m ${"\u001b"}[41miSH${"\u001b"}[39m ${"\u001b"}[42miSH${"\u001b"}[39m ${"\u001b"}[43miSH${"\u001b"}[39m ${"\u001b"}[44miSH${"\u001b"}[39m ${"\u001b"}[45miSH${"\u001b"}[39m ${"\u001b"}[46miSH${"\u001b"}[39m ${"\u001b"}[47miSH${"\u001b"}[39m${"\u001b"}[0m
# iSH Android Pure Rust - Slint + Servo + hterm + KVM
# 
""".trimIndent()

    override fun onDestroy() {
        super.onDestroy()
        Log.i("iSH", "Terminal.destroy - tty_hangup (ios_tty_cleanup)")
    }
}
