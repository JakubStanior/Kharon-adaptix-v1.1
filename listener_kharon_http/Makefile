all:
	@ echo "    * Building listener_kharon_http plugin"
	@ mkdir -p dist
	@ GOEXPERIMENT=jsonv2,greenteagc go build -buildmode=plugin -ldflags="-s -w" -o ./dist/listener_kharon_http.so pl_lokycrypt.go pl_main.go pl_http.go pl_listener.go
	@ echo "      done..."

	@ echo "    * Copying listener_kharon_http assets to dist"
	@ cp config.json dist/config.json
	@ cp ax_config.axs dist/ax_config.axs
	@ echo "      done..."
