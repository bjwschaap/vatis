SUMMARY = "OS Metrics MQTT publisher written in Go"
HOMEPAGE = "https://github.com/bjwschaap/vatis"
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://src/${GO_IMPORT}/LICENSE;md5=cd311743c0d91e20d206532af9d2d313"

GO_IMPORT = "github.com/bjwschaap/vatis"
SRC_URI = "git://${GO_IMPORT}"

# Points to 0.1.0 tag
SRCREV = "7a91b794bbfbf1f3b8b79823799316451127801b"

inherit go

GO_INSTALL = "${GO_IMPORT}"

RDEPENDS_${PN}-dev += "bash"
