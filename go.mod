module github.com/bincooo/emit.io

require (
	github.com/RomiChan/websocket v1.4.3-0.20220227141055-9b2c6168c9c5
	github.com/wangluozhe/chttp v0.0.4
	github.com/wangluozhe/requests v1.2.4
	golang.org/x/net v0.25.0
)

replace github.com/wangluozhe/requests v1.2.4 => github.com/bincooo/requests v0.0.0-20240627071243-53dc71945c20

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cloudflare/circl v1.3.8 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/refraction-networking/utls v1.6.6 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
)

go 1.21.6
