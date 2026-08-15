import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Page {
    id: page
    property color pageBackground: "#f2f2f7"
    property color pageForeground: "#1c1c1e"
    signal closeRequested()
    signal navigateRequested(string pageName)
    signal editRequested(string themeName)
    signal bootRootRequested()
    background: Rectangle { color: page.pageBackground }
}
