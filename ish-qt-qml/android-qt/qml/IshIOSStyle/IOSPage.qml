import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Page {
    id: page
    property color pageBackground: IOSPalette.surface("#f2f2f7")
    property color pageForeground: IOSPalette.text(page.pageBackground)
    property real contentInset: IOSMetrics.sideInset(width)
    property real groupedRowHeight: IOSMetrics.groupedRowHeight
    property real sectionHeaderHeight: IOSMetrics.sectionHeaderHeight
    property real sectionFooterHeight: IOSMetrics.sectionFooterHeight
    signal closeRequested()
    signal navigateRequested(string pageName)
    signal editRequested(string themeName)
    signal bootRootRequested()
    background: Rectangle { color: page.pageBackground }
}
