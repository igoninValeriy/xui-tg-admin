package services

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/pkg/xrayclient"
)

// XrayService manages X-ray API client for a single server
type XrayService struct {
	client *xrayclient.Client
	config *config.Config
	logger *logrus.Logger
}

// NewXrayService creates a new X-ray service
func NewXrayService(cfg *config.Config, logger *logrus.Logger) *XrayService {
	client := xrayclient.NewClient(cfg.Server, logger)

	return &XrayService{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// GetInbounds gets the inbounds from the server
func (s *XrayService) GetInbounds(ctx context.Context) ([]models.Inbound, error) {
	return s.client.GetInbounds(ctx)
}

// AddClient adds a client to an inbound on the server
func (s *XrayService) AddClient(ctx context.Context, inboundID int, client models.Client) error {
	return s.client.AddClientToInbound(ctx, inboundID, client)
}

// RemoveClients removes clients from the server
func (s *XrayService) RemoveClients(ctx context.Context, emails []string) error {
	return s.client.RemoveClients(ctx, emails)
}

// GetOnlineUsers gets the online users from the server
func (s *XrayService) GetOnlineUsers(ctx context.Context) ([]string, error) {
	return s.client.GetOnlineUsers(ctx)
}

// ResetUserTraffic resets a user's traffic on the server
func (s *XrayService) ResetUserTraffic(ctx context.Context, inboundID int, email string) error {
	return s.client.ResetUserTraffic(ctx, inboundID, email)
}

// GetAllMembersWithInfo получает детальную информацию о всех пользователях с поддержкой сортировки
func (s *XrayService) GetAllMembersWithInfo(ctx context.Context, sortType models.SortType) ([]models.MemberInfo, error) {
	inbounds, err := s.GetInbounds(ctx)
	if err != nil {
		return nil, err
	}

	// Группируем по SubID (общий для всех протоколов одного юзера), fallback на
	// базовое имя. Так клиенты одного пользователя в разных инбаундах сливаются
	// в одну запись, не полагаясь на разбор имени.
	emailToSubID := helpers.BuildEmailToSubID(inbounds)

	// Карта группировки пользователей по ключу группы
	memberMap := make(map[string]*models.MemberInfo)

	// Собираем информацию из ClientStats
	for _, inbound := range inbounds {
		for _, clientStat := range inbound.ClientStats {
			groupKey := helpers.UserGroupKey(clientStat.Email, emailToSubID)

			if memberInfo, exists := memberMap[groupKey]; exists {
				// Обновляем существующую запись
				memberInfo.FullEmails = append(memberInfo.FullEmails, clientStat.Email)
				memberInfo.TotalUp += clientStat.Up
				memberInfo.TotalDown += clientStat.Down
				memberInfo.TotalTraffic += clientStat.Up + clientStat.Down

				// Обновляем статус и время истечения
				if clientStat.Enable {
					memberInfo.Enable = true
				}
				if clientStat.ExpiryTime > memberInfo.ExpiryTime {
					memberInfo.ExpiryTime = clientStat.ExpiryTime
				}
				// Используем наименьший ID для сортировки по порядку создания
				if clientStat.ID < memberInfo.ID {
					memberInfo.ID = clientStat.ID
				}
			} else {
				// Создаем новую запись
				memberInfo := &models.MemberInfo{
					BaseUsername: helpers.ExtractBaseUsername(clientStat.Email),
					FullEmails:   []string{clientStat.Email},
					ID:           clientStat.ID,
					Enable:       clientStat.Enable,
					ExpiryTime:   clientStat.ExpiryTime,
					TotalUp:      clientStat.Up,
					TotalDown:    clientStat.Down,
					TotalTraffic: clientStat.Up + clientStat.Down,
				}
				memberMap[groupKey] = memberInfo
			}
		}
	}

	// Получаем дополнительную информацию из InboundSettings для каждого пользователя
	for _, inbound := range inbounds {
		if inbound.Settings == "" {
			continue
		}

		var settings models.InboundSettings
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}

		for _, client := range settings.Clients {
			groupKey := helpers.UserGroupKey(client.Email, emailToSubID)
			if memberInfo, exists := memberMap[groupKey]; exists {
				// Обновляем время истечения из настроек, если оно больше
				if client.ExpiryTime > memberInfo.ExpiryTime {
					memberInfo.ExpiryTime = client.ExpiryTime
				}
			}
		}
	}

	// Преобразуем карту в срез
	var members []models.MemberInfo
	for _, memberInfo := range memberMap {
		memberInfo.IsExpired = memberInfo.IsExpiredMember()
		members = append(members, *memberInfo)
	}

	// Сортируем по указанному типу
	models.SortMembers(members, sortType)

	return members, nil
}
