#!/usr/bin/env bash

set -euo pipefail

declare text=""
declare output=""
declare indicator=""

declare color_bg="0x663399"
declare color_fg="white"
declare font=""

while (("$#")); do
  case "${1}" in
  --bg)
    color_bg="${2}"
    shift 2
    ;;
  --fg)
    color_fg="${2}"
    shift 2
    ;;
  --font)
    font="${2}"
    shift 2
    ;;
  --text)
    text="${2}"
    shift 2
    ;;
  --output)
    output="${2}"
    shift 2
    ;;
  --indicator)
    indicator="${2}"
    shift 2
    ;;
  *)
    shift
    ;;
  esac
done

if test -z "${text}"; then
  echo "missing flag: --text"
  exit 1
fi

if test -z "${output}"; then
  echo "missing flag: --output"
  exit 1
fi

if test -z "${font}"; then
  font="${TMPDIR}Excalifont-Regular.ttf"
  if ! test -e "${font}"; then
    declare -r font_url="$(curl -fsSL "https://plus.excalidraw.com/excalifont" | grep -o 'href="[^"]*Excalifont-Regular.woff2"' | head -1 | cut -d'=' -f2 | sed -e 's/"//g')"
    declare -r font_path="${TMPDIR}Excalifont-Regular.woff2"
    curl -o "${font_path}" -fsSL "${font_url}"
    fontforge -lang=ff -c 'Open("'${font_path}'"); Generate("'${font}'")'
  fi
fi

declare -r size="1280x720"
declare rate="1"
declare duration="30"

declare -r fontfile="${font}"
declare -r fontsize="48"

declare -r opt_x="x=(w-text_w)/2"
declare -r opt_y="y=(h-text_h)/2"

declare -r common_opts="fontfile=${fontfile}:fontsize=${fontsize}:fontcolor=${color_fg}:${opt_x}"

declare content="
  drawtext=text='[StremThru]':${common_opts}:${opt_y}-60,
  drawtext=text='${text}':${common_opts}:${opt_y},
"

if test -n "${indicator}"; then
  declare total_indicator_frame="$(echo "${indicator}" | tr '|' '\n' | wc -l)"
  declare indicator_frame=""
  for ((i = 0; i < total_indicator_frame; i++)); do
    indicator_frame="$(echo "${indicator}" | cut -d'|' -f$((i + 1)))"
    content="${content}
    drawtext=text='${indicator_frame}':${common_opts}:${opt_y}+60:enable='eq(mod(t,$total_indicator_frame),$i)',"
  done
fi

ffmpeg -f lavfi -i color=color=${color_bg}:size=${size}:rate=1 -vf "${content}" -t ${duration} -y ${output}
