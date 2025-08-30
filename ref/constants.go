package main


var supportedEventTypes = []string{

	"Message",
	"UndecryptableMessage",
	"Receipt",
	"MediaRetry",
	"ReadReceipt",


	"GroupInfo",
	"JoinedGroup",
	"Picture",
	"BlocklistChange",
	"Blocklist",


	"Connected",
	"Disconnected",
	"ConnectFailure",
	"KeepAliveRestored",
	"KeepAliveTimeout",
	"LoggedOut",
	"ClientOutdated",
	"TemporaryBan",
	"StreamError",
	"StreamReplaced",
	"PairSuccess",
	"PairError",
	"QR",
	"QRScannedWithoutMultidevice",


	"PrivacySettings",
	"PushNameSetting",
	"UserAbout",


	"AppState",
	"AppStateSyncComplete",
	"HistorySync",
	"OfflineSyncCompleted",
	"OfflineSyncPreview",


	"CallOffer",
	"CallAccept",
	"CallTerminate",
	"CallOfferNotice",
	"CallRelayLatency",


	"Presence",
	"ChatPresence",


	"IdentityChange",


	"CATRefreshError",


	"NewsletterJoin",
	"NewsletterLeave",
	"NewsletterMuteChange",
	"NewsletterLiveUpdate",


	"FBMessage",


	"All",
}


var eventTypeMap map[string]bool

func init() {
	eventTypeMap = make(map[string]bool)
	for _, eventType := range supportedEventTypes {
		eventTypeMap[eventType] = true
	}
}


func isValidEventType(eventType string) bool {
	return eventTypeMap[eventType]
}
