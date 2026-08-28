import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Button {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitHeight: IOSMetrics.groupedRowHeight
    padding: 8
    font.pixelSize: IOSMetrics.rowLabelSize
    contentItem: Text {
        text: control.text
        color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor)
        font: control.font
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
    }
    background: Rectangle {
        // Rounded clipped batches are corrupted by some Android emulator RHI
        // configurations; use flat controls there while preserving iOS styling
        // on desktop and other platforms.
        radius: Qt.platform.os === "android" ? 0 : IOSMetrics.controlCornerRadius
        layer.enabled: Qt.platform.os === "android"
        layer.smooth: true
        color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : IOSPalette.elevatedSurface(control.styleWindowColor)
        border.width: Qt.platform.os === "android" ? 0 : 1
        border.color: IOSPalette.separator(control.styleWindowColor)
        opacity: control.enabled ? 1 : .55
    }
}
