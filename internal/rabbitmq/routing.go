package rabbitmq

// RoutingKeyForJob возвращает routing key для topic exchange.
func RoutingKeyForJob(msg *AvatarJobMessage) string {
	switch msg.Type {
	case AvatarJobTypeProcess:
		return RoutingKeyAvatarProcess
	case AvatarJobTypeDeleteByS3Key:
		return RoutingKeyAvatarDelete
	default:
		return ""
	}
}
