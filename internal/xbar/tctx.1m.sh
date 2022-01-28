#!/usr/bin/env bash

# <xbar.title>Temporal</xbar.title>
# <xbar.version>v1.0</xbar.version>
# <xbar.author>Jacob LeGrone</xbar.author>
# <xbar.author.github>jlegrone</xbar.author.github>
# <xbar.desc>Switch Temporal cluster and namespace contexts.</xbar.desc>
# <xbar.abouturl>https://github.com/jlegrone/tctx</xbar.abouturl>
# <xbar.image>https://github.com/jlegrone/tctx/raw/jlegrone/xbar/internal/xbar/screenshot.png</xbar.image>
# <xbar.dependencies>tctx,tctl</xbar.dependencies>
# <xbar.var>boolean(SHOW_CLUSTER=""): Display Temporal cluster name in menu bar.</xbar.var>
# <xbar.var>boolean(SHOW_NAMESPACE=""): Display Temporal namespace in menu bar.</xbar.var>
# <xbar.var>string(TCTX_BIN="tctx"): Path to tctx executable.</xbar.var>
# <xbar.var>string(TCTL_BIN="tctl"): Path to tctl executable.</xbar.var>

export PATH="/usr/local/bin:/usr/bin:$PATH";

# Set defaults again just in case they were deleted in plugin settings:
export TCTX_BIN="${TCTX_BIN:-tctx}";
export TCTL_BIN="${TCTL_BIN:-tctl}";

# Render menu items
"$TCTX_BIN" tctxbar
