pragma Singleton
import QtQuick

QtObject {
    // Logical points from the original iSH iOS storyboards. Qt Quick uses
    // logical coordinates, so these values remain stable across DPI values.
    readonly property real navigationBarHeight: 44
    readonly property real accessoryBarHeight: 50
    readonly property real accessoryButtonWidth: 31
    readonly property real accessoryTouchHeight: 50
    readonly property real accessoryOuterInset: 6
    readonly property real accessorySpacing: 6

    readonly property real groupedRowHeight: 43.5
    readonly property real groupedDetailRowHeight: 55.5
    readonly property real sectionHeaderHeight: 18
    readonly property real sectionFooterHeight: 18
    readonly property real tableHorizontalInset: 16
    readonly property real tableWideHorizontalInset: 20
    readonly property real rowLabelSize: 17
    readonly property real rowDetailSize: 14
    readonly property real sectionLabelSize: 13
    readonly property real footerLabelSize: 13
    readonly property real navigationTitleSize: 17
    readonly property real navigationButtonSize: 17
    readonly property real switchWidth: 51
    readonly property real switchHeight: 31
    readonly property real minimumTouchTarget: 44
    readonly property real controlCornerRadius: 9
    readonly property real groupedCornerRadius: 10

    function sideInset(width) {
        return width >= 600 ? tableWideHorizontalInset : tableHorizontalInset
    }
}
