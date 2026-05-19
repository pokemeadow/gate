package pokeperms

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.minekube.com/brigodier"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/common/minecraft/component/codec/legacy"
	"go.minekube.com/gate/pkg/command"
	"go.minekube.com/gate/pkg/edition/java/proxy"
	"go.minekube.com/gate/pkg/util/uuid"
)

func colorText(s string) component.Component {
	leg := legacy.Legacy{
		Char:    legacy.AmpersandChar,
		HexChar: legacy.HexChar,
	}
	comp, err := leg.Unmarshal([]byte(s))
	if err != nil {
		return &component.Text{Content: s}
	}
	return comp
}

func RegisterCommands(p *proxy.Proxy, storage *Storage, cfg *Config, msgMgr *MessagingManager) {
	h := &commandHandler{
		proxy:   p,
		storage: storage,
		cfg:     cfg,
		msgMgr:  msgMgr,
	}

	root := brigodier.Literal("pokeperms").
		Requires(command.Requires(func(c *command.RequiresContext) bool {
			_, isPlayer := c.Source.(proxy.Player)
			if !isPlayer {
				return true
			}

			hasPerm := c.Source.HasPermission("pokeperms.gate.admin")
			if !hasPerm {
				c.Source.SendMessage(colorText(cfg.Messages.Prefix + cfg.Messages.NoPermission))
			}
			return hasPerm
		})).
		Executes(command.Command(func(c *command.Context) error {
			c.Source.SendMessage(colorText(cfg.Messages.Prefix + "&7Version 1.0.0 - Use /pp user or /pp group"))
			return nil
		}))

	userNode := brigodier.Literal("user").
		Then(brigodier.Argument("player", brigodier.String).
			Then(brigodier.Literal("info").Executes(command.Command(h.userInfo))).
			Then(brigodier.Literal("clear").Executes(command.Command(h.userClear))).
			Then(brigodier.Literal("parent").
				Then(brigodier.Literal("set").
					Then(brigodier.Argument("group", brigodier.String).
						Executes(command.Command(h.userParentSet)))).
				Then(brigodier.Literal("addtemp").
					Then(brigodier.Argument("group", brigodier.String).
						Then(brigodier.Argument("duration", brigodier.String).
							Executes(command.Command(h.userParentAddTemp)))))).
			Then(brigodier.Literal("permission").
				Then(brigodier.Literal("set").
					Then(brigodier.Argument("node", brigodier.String).
						Then(brigodier.Argument("value", brigodier.Bool).
							Executes(command.Command(h.userPermSet)).
							Then(brigodier.Argument("server", brigodier.String).
								Executes(command.Command(h.userPermSet)))))).
				Then(brigodier.Literal("unset").
					Then(brigodier.Argument("node", brigodier.String).
						Executes(command.Command(h.userPermUnset)).
						Then(brigodier.Argument("server", brigodier.String).
							Executes(command.Command(h.userPermUnset)))))))

	groupNode := brigodier.Literal("group").
		Then(brigodier.Argument("group_name", brigodier.String).
			Then(brigodier.Literal("create").Executes(command.Command(h.groupCreate))).
			Then(brigodier.Literal("delete").Executes(command.Command(h.groupDelete))).
			Then(brigodier.Literal("info").Executes(command.Command(h.groupInfo))).
			Then(brigodier.Literal("prefix").
				Then(brigodier.Literal("set").
					Then(brigodier.Argument("prefix_str", brigodier.String).
						Executes(command.Command(h.groupPrefix))))).
			Then(brigodier.Literal("setweight").
				Then(brigodier.Argument("weight", brigodier.Int).
					Executes(command.Command(h.groupWeight)))).
			Then(brigodier.Literal("parent").
				Then(brigodier.Literal("add").
					Then(brigodier.Argument("parent_name", brigodier.String).Executes(command.Command(h.groupParentAdd)))).
				Then(brigodier.Literal("remove").
					Then(brigodier.Argument("parent_name", brigodier.String).Executes(command.Command(h.groupParentRemove))))).
			Then(brigodier.Literal("permission").
				Then(brigodier.Literal("set").
					Then(brigodier.Argument("node", brigodier.String).
						Then(brigodier.Argument("value", brigodier.Bool).
							Executes(command.Command(h.groupPermSet)).
							Then(brigodier.Argument("server", brigodier.String).
								Executes(command.Command(h.groupPermSet)))))).
				Then(brigodier.Literal("unset").
					Then(brigodier.Argument("node", brigodier.String).
						Executes(command.Command(h.groupPermUnset)).
						Then(brigodier.Argument("server", brigodier.String).
							Executes(command.Command(h.groupPermUnset)))))))

	reloadNode := brigodier.Literal("reload").Executes(command.Command(h.handleReload))
	syncNode := brigodier.Literal("sync").Executes(command.Command(h.handleSync))

	cmd := root.Then(userNode).Then(groupNode).Then(reloadNode).Then(syncNode)
	p.Command().RegisterWithAliases(cmd, cfg.Aliases...)
}

type commandHandler struct {
	proxy   *proxy.Proxy
	storage *Storage
	cfg     *Config
	msgMgr  *MessagingManager
}

func (h *commandHandler) msg(s string) string {
	return h.cfg.Messages.Prefix + s
}

func (h *commandHandler) resolvePlayerUUID(username string) string {
	player := h.proxy.PlayerByName(username)
	if player != nil {
		return player.ID().Undashed()
	}
	return uuid.OfflinePlayerUUID(username).Undashed()
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func optionalServer(c *command.Context) string {
	val := c.String("server")
	if val == "" {
		return "global"
	}
	return val
}

func (h *commandHandler) userInfo(c *command.Context) error {
	username := c.String("player")
	uid := h.resolvePlayerUUID(username)

	var primaryGroup string
	err := h.storage.db.QueryRow("SELECT primary_group FROM pp_users WHERE uuid = ?", uid).Scan(&primaryGroup)
	if err == sql.ErrNoRows {
		primaryGroup = "default"
	} else if err != nil {
		c.Source.SendMessage(colorText(h.msg("&cDatabase Error: " + err.Error())))
		return nil
	}

	c.Source.SendMessage(colorText(h.msg("&b&l=== User Profile: " + username + " ===")))
	c.Source.SendMessage(colorText(h.msg("&aPrimary Group: &e" + primaryGroup)))

	rows, err := h.storage.db.Query("SELECT group_name, expiry_time FROM pp_user_temp_groups WHERE uuid = ?", uid)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gName string
			var expiry time.Time
			if err := rows.Scan(&gName, &expiry); err == nil {
				timeLeft := time.Until(expiry).Round(time.Second)
				if timeLeft > 0 {
					c.Source.SendMessage(colorText(h.msg(fmt.Sprintf("&aTemp Group: &e%s &7(Expires in: %s)", gName, timeLeft))))
				}
			}
		}
	}

	pRows, err := h.storage.db.Query("SELECT permission, value, server FROM pp_user_permissions WHERE uuid = ?", uid)
	if err == nil {
		defer pRows.Close()
		c.Source.SendMessage(colorText(h.msg("&aCustom Permissions:")))
		hasPerms := false
		for pRows.Next() {
			var perm, server string
			var val int
			if err := pRows.Scan(&perm, &val, &server); err == nil {
				hasPerms = true
				status := "&a✓"
				if val == 0 {
					status = "&c✗"
				}
				c.Source.SendMessage(colorText(h.msg(fmt.Sprintf("  %s &7%s &8(Server: %s)", status, perm, server))))
			}
		}
		if !hasPerms {
			c.Source.SendMessage(colorText(h.msg("  &7None")))
		}
	}
	return nil
}

func (h *commandHandler) userClear(c *command.Context) error {
	username := c.String("player")
	uid := h.resolvePlayerUUID(username)

	if err := h.storage.ClearUser(uid); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("USER_UPDATE", uid)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.UserCleared, username))))
	return nil
}

func (h *commandHandler) userParentSet(c *command.Context) error {
	username := c.String("player")
	group := c.String("group")
	uid := h.resolvePlayerUUID(username)

	h.storage.EnsureUser(uid, username)
	if err := h.storage.SetUserGroup(uid, group); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("USER_UPDATE", uid)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.UserParentSet, username, group))))
	return nil
}

func (h *commandHandler) userParentAddTemp(c *command.Context) error {
	username := c.String("player")
	group := c.String("group")
	durStr := c.String("duration")
	uid := h.resolvePlayerUUID(username)

	duration, err := parseDuration(durStr)
	if err != nil {
		c.Source.SendMessage(colorText(h.msg(h.cfg.Messages.InvalidDuration)))
		return nil
	}

	h.storage.EnsureUser(uid, username)
	if err := h.storage.AddUserTempGroup(uid, group, duration); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("USER_UPDATE", uid)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.UserParentAddTemp, group, username, durStr))))
	return nil
}

func (h *commandHandler) userPermSet(c *command.Context) error {
	username := c.String("player")
	node := c.String("node")
	val := c.Bool("value")
	server := optionalServer(c)
	uid := h.resolvePlayerUUID(username)

	h.storage.EnsureUser(uid, username)
	if err := h.storage.SetUserPermission(uid, node, val, server); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("USER_UPDATE", uid)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.UserPermSet, node, val, username, server))))
	return nil
}

func (h *commandHandler) userPermUnset(c *command.Context) error {
	username := c.String("player")
	node := c.String("node")
	server := optionalServer(c)
	uid := h.resolvePlayerUUID(username)

	if err := h.storage.UnsetUserPermission(uid, node, server); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("USER_UPDATE", uid)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.UserPermUnset, node, username, server))))
	return nil
}

func (h *commandHandler) groupCreate(c *command.Context) error {
	group := c.String("group_name")
	if err := h.storage.CreateGroup(group); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupCreated, group))))
	return nil
}

func (h *commandHandler) groupDelete(c *command.Context) error {
	group := c.String("group_name")
	if err := h.storage.DeleteGroup(group); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupDeleted, group))))
	return nil
}

func (h *commandHandler) groupInfo(c *command.Context) error {
	group := c.String("group_name")

	var prefix string
	var weight int
	err := h.storage.db.QueryRow("SELECT prefix, weight FROM pp_groups WHERE name = ?", group).Scan(&prefix, &weight)
	if err == sql.ErrNoRows {
		c.Source.SendMessage(colorText(h.msg("&cError: Group does not exist.")))
		return nil
	} else if err != nil {
		c.Source.SendMessage(colorText(h.msg("&cDatabase Error: " + err.Error())))
		return nil
	}

	c.Source.SendMessage(colorText(h.msg("&b&l=== Group Data: " + group + " ===")))
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf("&aPrefix: &f%s &7(Raw: %s)", prefix, prefix))))
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf("&aWeight: &e%d", weight))))

	rows, err := h.storage.db.Query("SELECT parent_name FROM pp_group_parents WHERE group_name = ?", group)
	if err == nil {
		defer rows.Close()
		var parents []string
		for rows.Next() {
			var pName string
			if err := rows.Scan(&pName); err == nil {
				parents = append(parents, pName)
			}
		}
		c.Source.SendMessage(colorText(h.msg("&aInherited Groups: &e" + strings.Join(parents, ", "))))
	}

	pRows, err := h.storage.db.Query("SELECT permission, value, server FROM pp_group_permissions WHERE group_name = ?", group)
	if err == nil {
		defer pRows.Close()
		c.Source.SendMessage(colorText(h.msg("&aGroup Permissions:")))
		hasPerms := false
		for pRows.Next() {
			var perm, server string
			var val int
			if err := pRows.Scan(&perm, &val, &server); err == nil {
				hasPerms = true
				status := "&a✓"
				if val == 0 {
					status = "&c✗"
				}
				c.Source.SendMessage(colorText(h.msg(fmt.Sprintf("  %s &7%s &8(Server: %s)", status, perm, server))))
			}
		}
		if !hasPerms {
			c.Source.SendMessage(colorText(h.msg("  &7None")))
		}
	}
	return nil
}

func (h *commandHandler) groupPrefix(c *command.Context) error {
	group := c.String("group_name")
	prefix := c.String("prefix_str")
	if err := h.storage.SetGroupPrefix(group, prefix); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupPrefixSet, group, prefix))))
	return nil
}

func (h *commandHandler) groupWeight(c *command.Context) error {
	group := c.String("group_name")
	weight := c.Int("weight")
	if err := h.storage.SetGroupWeight(group, weight); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupWeightSet, group, weight))))
	return nil
}

func (h *commandHandler) groupParentAdd(c *command.Context) error {
	group := c.String("group_name")
	parent := c.String("parent_name")
	if err := h.storage.AddGroupParent(group, parent); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupParentAdded, group, parent))))
	return nil
}

func (h *commandHandler) groupParentRemove(c *command.Context) error {
	group := c.String("group_name")
	parent := c.String("parent_name")
	if err := h.storage.RemoveGroupParent(group, parent); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupParentRemoved, group, parent))))
	return nil
}

func (h *commandHandler) groupPermSet(c *command.Context) error {
	group := c.String("group_name")
	node := c.String("node")
	val := c.Bool("value")
	server := optionalServer(c)

	if err := h.storage.SetGroupPermission(group, node, val, server); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupPermSet, node, val, group, server))))
	return nil
}

func (h *commandHandler) groupPermUnset(c *command.Context) error {
	group := c.String("group_name")
	node := c.String("node")
	server := optionalServer(c)

	if err := h.storage.UnsetGroupPermission(group, node, server); err != nil {
		c.Source.SendMessage(colorText(h.msg("&cError: " + err.Error())))
		return nil
	}
	_ = h.msgMgr.BroadcastUpdate("GROUP_UPDATE", group)
	c.Source.SendMessage(colorText(h.msg(fmt.Sprintf(h.cfg.Messages.GroupPermUnset, node, group, server))))
	return nil
}

func (h *commandHandler) handleReload(c *command.Context) error {
	c.Source.SendMessage(colorText(h.msg("&eReloading configurations and localized messages...")))
	dir := fmt.Sprintf("plugins/%s", PluginName)
	newCfg, err := LoadConfig(dir)
	if err != nil {
		c.Source.SendMessage(colorText(h.msg("&cReload Failed! Error: " + err.Error())))
		return nil
	}
	*h.cfg = *newCfg
	c.Source.SendMessage(colorText(h.msg("&aPlugin configurations successfully reloaded!")))
	return nil
}

func (h *commandHandler) handleSync(c *command.Context) error {
	c.Source.SendMessage(colorText(h.msg("&eSyncing active database structures to all backend servers...")))
	err := h.msgMgr.BroadcastUpdate("SYNC_ALL", "global")
	if err != nil {
		c.Source.SendMessage(colorText(h.msg("&cNetwork Sync Failed: " + err.Error())))
		return nil
	}
	c.Source.SendMessage(colorText(h.msg("&aNetwork sync request successfully dispatched to all online nodes!")))
	return nil
}