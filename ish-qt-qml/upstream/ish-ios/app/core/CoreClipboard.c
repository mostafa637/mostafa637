/*
 * Qt port compatibility unit.
 *
 * The original iSH iOS target exposes clipboard callbacks from Objective-C.
 * This port deliberately keeps the core free of UIKit: clipboard access is
 * provided by PlatformServicesQt through QClipboard. The empty translation
 * unit remains in the core tree so source-completeness checks and historical
 * build manifests retain the original layout.
 */
