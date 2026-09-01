package models

import (
	"fmt"
	"sort"
	"time"
)

// SortType представляет тип сортировки пользователей
type SortType int

const (
	SortByCreationOrder SortType = iota // По порядку создания (ID)
	SortByExpiryDate                    // По дате истечения
	SortByTrafficTotal                  // По общему трафику
	SortByStatus                        // По статусу (активные первые)
	SortByName                          // По имени (алфавитный)
	SortByLastOnline                    // По времени последнего подключения (свежие первые)
)

// MemberInfo содержит расширенную информацию о пользователе для сортировки и фильтрации
type MemberInfo struct {
	BaseUsername string   // Базовое имя пользователя (без постфикса)
	FullEmails   []string // Все email'ы пользователя во всех inbound'ах
	SubID        string   // Идентификатор подписки (общий для всех клиентов юзера)
	ID           int      // ID для сортировки по порядку создания
	Enable       bool     // Активен ли пользователь
	ExpiryTime   int64    // Время истечения (миллисекунды)
	TotalUp      int64    // Общий загруженный трафик
	TotalDown    int64    // Общий скачанный трафик
	TotalTraffic int64    // Общий трафик (Up + Down)
	LastOnline   int64    // Последнее подключение (миллисекунды), 0 — не подключался ни разу
	IsExpired    bool     // Истек ли срок действия
}

// LastSeenStatus возвращает читаемое время последнего подключения. Панель 3.7.0
// отдаёт его в /panel/api/clients/lastOnline и в traffic.lastOnline; ноль
// означает, что клиент не подключался ни разу.
func (m *MemberInfo) LastSeenStatus() string {
	if m.LastOnline == 0 {
		return "никогда"
	}

	elapsed := time.Since(time.UnixMilli(m.LastOnline))
	switch {
	case elapsed < time.Minute:
		return "только что"
	case elapsed < time.Hour:
		return fmt.Sprintf("%d мин. назад", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%d ч. назад", int(elapsed.Hours()))
	default:
		return fmt.Sprintf("%d дн. назад", int(elapsed.Hours()/24))
	}
}

// IsExpiredMember проверяет, истек ли срок действия пользователя
func (m *MemberInfo) IsExpiredMember() bool {
	if m.ExpiryTime == 0 {
		return false // Бессрочный
	}
	return time.Now().UnixMilli() > m.ExpiryTime
}

// GetExpiryStatus возвращает статус истечения в читаемом виде
func (m *MemberInfo) GetExpiryStatus() string {
	if m.ExpiryTime == 0 {
		return "∞ Бессрочный"
	}

	if m.IsExpiredMember() {
		return "❌ Истек"
	}

	expiryDate := time.Unix(m.ExpiryTime/1000, 0)
	daysLeft := int(time.Until(expiryDate).Hours() / 24)

	if daysLeft <= 0 {
		return "⚠️ Истекает сегодня"
	} else if daysLeft <= 7 {
		return fmt.Sprintf("⚠️ %d дн.", daysLeft)
	}

	return fmt.Sprintf("✅ %d дн.", daysLeft)
}

// SortMembers сортирует список пользователей по указанному типу
func SortMembers(members []MemberInfo, sortType SortType) {
	sort.Slice(members, func(i, j int) bool {
		switch sortType {
		case SortByCreationOrder:
			return members[i].ID < members[j].ID
		case SortByExpiryDate:
			// Бессрочные в конец, остальные по возрастанию даты истечения
			if members[i].ExpiryTime == 0 && members[j].ExpiryTime == 0 {
				return members[i].BaseUsername < members[j].BaseUsername
			}
			if members[i].ExpiryTime == 0 {
				return false
			}
			if members[j].ExpiryTime == 0 {
				return true
			}
			return members[i].ExpiryTime < members[j].ExpiryTime
		case SortByTrafficTotal:
			return members[i].TotalTraffic > members[j].TotalTraffic // По убыванию
		case SortByStatus:
			// Активные первые, потом неактивные
			if members[i].Enable != members[j].Enable {
				return members[i].Enable
			}
			return members[i].BaseUsername < members[j].BaseUsername
		case SortByName:
			return members[i].BaseUsername < members[j].BaseUsername
		case SortByLastOnline:
			// Никогда не подключавшиеся уходят в конец списка
			if members[i].LastOnline == members[j].LastOnline {
				return members[i].BaseUsername < members[j].BaseUsername
			}
			return members[i].LastOnline > members[j].LastOnline
		default:
			return members[i].ID < members[j].ID
		}
	})
}
