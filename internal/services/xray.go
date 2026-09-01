package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/pkg/xrayclient"
)

// XrayService wraps the 3x-ui API client with the aggregation the bot needs.
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

// GetInbounds gets the inbounds from the panel
func (s *XrayService) GetInbounds(ctx context.Context) ([]models.Inbound, error) {
	return s.client.GetInbounds(ctx)
}

// GetEnabledInboundIDs returns the IDs of every enabled inbound. Since 3.7.0 a
// client is attached to a list of inbounds in a single call, so these IDs are
// what client creation needs rather than one request per inbound.
func (s *XrayService) GetEnabledInboundIDs(ctx context.Context) ([]int, error) {
	inbounds, err := s.GetInbounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbounds: %w", err)
	}

	ids := make([]int, 0, len(inbounds))
	for _, inbound := range inbounds {
		if inbound.Enable {
			ids = append(ids, inbound.ID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no enabled inbounds available")
	}
	return ids, nil
}

// AddClient creates a client and attaches it to the given inbounds.
func (s *XrayService) AddClient(ctx context.Context, client models.Client, inboundIDs []int) error {
	return s.client.AddClient(ctx, client, inboundIDs)
}

// RemoveClients removes clients from the panel
func (s *XrayService) RemoveClients(ctx context.Context, emails []string) error {
	return s.client.RemoveClients(ctx, emails)
}

// GetOnlineUsers returns the emails of clients connected within the panel's
// heartbeat window.
func (s *XrayService) GetOnlineUsers(ctx context.Context) ([]string, error) {
	return s.client.GetOnlineUsers(ctx)
}

// GetLastOnline maps client email to when it was last seen (Unix ms).
func (s *XrayService) GetLastOnline(ctx context.Context) (map[string]int64, error) {
	return s.client.GetLastOnline(ctx)
}

// GetClientLinks returns the panel's ready-made share URLs for one client.
func (s *XrayService) GetClientLinks(ctx context.Context, email string) ([]string, error) {
	return s.client.GetClientLinks(ctx, email)
}

// ResetClientTraffic resets one client's traffic across every inbound.
func (s *XrayService) ResetClientTraffic(ctx context.Context, email string) error {
	return s.client.ResetClientTraffic(ctx, email)
}

// ResetAllTraffics resets the traffic counters of every client on the panel.
func (s *XrayService) ResetAllTraffics(ctx context.Context) error {
	return s.client.ResetAllTraffics(ctx)
}

// SubscriptionURL builds the subscription URL for a subscription ID. The
// subscription is served by the configured front end (a Cloudflare Worker in
// this deployment), not by the panel, so the URL is just the prefix joined with
// the ID.
func (s *XrayService) SubscriptionURL(subID string) string {
	if subID == "" || s.config.Server.SubURLPrefix == "" {
		return ""
	}
	return strings.TrimRight(s.config.Server.SubURLPrefix, "/") + "/" + subID
}

// FindMemberClients returns the client records whose email belongs to the given
// base username. Clients created before 3.7.0 carry a "-N" inbound suffix, so
// one logical user can still map to several records.
func (s *XrayService) FindMemberClients(ctx context.Context, baseUsername string) ([]models.ClientRecord, error) {
	clients, err := s.client.GetClients(ctx)
	if err != nil {
		return nil, err
	}

	var matched []models.ClientRecord
	for _, record := range clients {
		if helpers.IsEmailMatchingBaseUsername(record.Email, baseUsername) {
			matched = append(matched, record)
		}
	}
	return matched, nil
}

// GetAllMembersWithInfo collapses the panel's clients into one entry per user
// and sorts them. It reads /panel/api/clients/list, which since 3.7.0 is the
// authoritative view of clients — the per-inbound settings blob only mirrors it.
func (s *XrayService) GetAllMembersWithInfo(ctx context.Context, sortType models.SortType) ([]models.MemberInfo, error) {
	clients, err := s.client.GetClients(ctx)
	if err != nil {
		return nil, err
	}

	// Group by SubID when present (a user's clients share one), falling back to
	// the base username so legacy "name-1"/"name-2" pairs still merge.
	emailToSubID := make(map[string]string, len(clients))
	for _, record := range clients {
		if record.SubID != "" {
			emailToSubID[record.Email] = record.SubID
		}
	}

	memberMap := make(map[string]*models.MemberInfo)
	order := make([]string, 0, len(clients))

	for _, record := range clients {
		groupKey := helpers.UserGroupKey(record.Email, emailToSubID)

		member, exists := memberMap[groupKey]
		if !exists {
			member = &models.MemberInfo{
				BaseUsername: helpers.ExtractBaseUsername(record.Email),
				ID:           record.ID,
				SubID:        record.SubID,
			}
			memberMap[groupKey] = member
			order = append(order, groupKey)
		}

		member.FullEmails = append(member.FullEmails, record.Email)
		member.TotalUp += record.Traffic.Up
		member.TotalDown += record.Traffic.Down
		member.TotalTraffic += record.Traffic.Up + record.Traffic.Down
		if record.Enable {
			member.Enable = true
		}
		if record.ExpiryTime > member.ExpiryTime {
			member.ExpiryTime = record.ExpiryTime
		}
		if record.Traffic.LastOnline > member.LastOnline {
			member.LastOnline = record.Traffic.LastOnline
		}
		if record.ID < member.ID {
			member.ID = record.ID
		}
		if member.SubID == "" {
			member.SubID = record.SubID
		}
	}

	members := make([]models.MemberInfo, 0, len(memberMap))
	for _, key := range order {
		member := memberMap[key]
		member.IsExpired = member.IsExpiredMember()
		members = append(members, *member)
	}

	models.SortMembers(members, sortType)
	return members, nil
}
