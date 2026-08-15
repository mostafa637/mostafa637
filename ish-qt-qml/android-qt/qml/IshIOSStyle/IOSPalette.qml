pragma Singleton
import QtQuick

QtObject {
    readonly property color blue: "#007aff"
    readonly property color green: "#34c759"
    readonly property color red: "#ff3b30"
    readonly property color orange: "#ff9500"
    readonly property color secondary: "#8e8e93"
    function isDark(background) { var c = Qt.color(background); return c.valid && (0.299*c.r + 0.587*c.g + 0.114*c.b) < 0.5 }
    function surface(background) { return isDark(background) ? "#1c1c1e" : "#f2f2f7" }
    function elevatedSurface(background) { return isDark(background) ? "#2c2c2e" : "#ffffff" }
    function text(background) { return isDark(background) ? "#f5f5f7" : "#1c1c1e" }
    function secondaryText(background) { return isDark(background) ? "#aeaeb2" : "#6c6c70" }
    function separator(background) { return isDark(background) ? "#48484a" : "#c6c6c8" }
}
