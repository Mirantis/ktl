{{- define "merge_presets" -}}
{{- $preset_values := .Values.preset_values -}}
{{- $global := .Values.global -}}
{{- range $idx, $preset := .Values.presets -}}
{{-   range $key, $value := index $preset_values $preset -}}
{{-     $_ := set $global $key $value -}}
{{-   end -}}
{{- end -}}
{{- end -}}
