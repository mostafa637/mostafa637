import QtQuick
import QtQuick.Controls as Controls

Controls.ScrollBar {
    id: bar
    property real contentY: 0
    property real contentHeight: 0
    property real viewportHeight: 0
    signal contentYRequested(real value)
    orientation: Qt.Vertical
    policy: Controls.ScrollBar.AsNeeded
    size: contentHeight > 0 ? Math.min(1, viewportHeight / contentHeight) : 1
    position: contentHeight > viewportHeight ? contentY / (contentHeight - viewportHeight) : 0
    onPositionChanged: if (pressed && contentHeight > viewportHeight) contentYRequested(position * (contentHeight - viewportHeight))
}
