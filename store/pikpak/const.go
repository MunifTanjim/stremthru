package pikpak

import "net/url"

const clientId = "YNxT9w7GMdWvEOKa"
const clientSecret = "dbw2OtmVEeuUvIptb1Coyg"
const clientVersion = "1.47.1"
const packageName = "com.pikcloud.pikpak"
const sdkVersion = "2.0.4.204000"

var DriveAPIBaseURL = func() *url.URL {
	u, err := url.Parse("https://api-drive.mypikpak.com")
	if err != nil {
		panic(err)
	}
	return u
}()

var UserAPIBaseURL = func() *url.URL {
	u, err := url.Parse("https://user.mypikpak.com")
	if err != nil {
		panic(err)
	}
	return u
}()
