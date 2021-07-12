package onecloud

import (
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"

	"yunion.io/x/kubecomps/pkg/types"
)

type DisksHelper struct {
	*ResourceHelper
}

func NewDisksHelper(s *mcclient.ClientSession) *DisksHelper {
	return &DisksHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Disks),
	}
}

func (c *DisksHelper) Get(diskName string) (*types.SDisk, error) {
	s := c.session
	diskObj, err := modules.Disks.Get(s, diskName, nil)
	if err != nil {
		return nil, err
	}
	disk := new(types.SDisk)
	err = diskObj.Unmarshal(disk)
	return disk, err
}

func (c *DisksHelper) Create(input *types.CreateDiskInput) (*types.SDisk, error) {
	s := c.session
	data := &computeapi.DiskCreateInput{
		DiskConfig: &computeapi.DiskConfig{
			SizeMb:  input.Size,
			Fs:      input.Fs,
			Format:  input.Format,
			Backend: input.Backend,
		},
		Name:       input.Name,
		PreferHost: input.PreferHost,
	}
	diskObj, err := modules.Disks.Create(s, data.JSON(data))
	if err != nil {
		return nil, errors.Errorf("Create disk %s: %v", data.Name, err)
	}
	disk := new(types.SDisk)
	err = diskObj.Unmarshal(disk)
	return disk, err
}

func (c *DisksHelper) DeleteDisk(id string) error {
	s := c.session
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	_, err := modules.Disks.DeleteWithParam(s, id, params, nil)
	return err
}

func continueWaitResource(status string) bool {
	if strings.HasSuffix(status, "_fail") || strings.HasSuffix(status, "_failed") {
		return false
	}
	return true
}

func (c *DisksHelper) WaitReady(id string) error {
	return c.WaitObjectStatus(
		id, computeapi.DISK_READY, continueWaitResource,
	)
}

func (c *DisksHelper) AttachToGuest(diskId string, guestId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(diskId), "disk_id")
	_, err := modules.Servers.PerformAction(c.session, guestId, "attachdisk", params)
	return err
}

func (c *DisksHelper) EnsureDetach(diskId string, guestId string) error {
	has, err := c.IsGuestHasDisk(diskId, guestId)
	if err != nil {
		return errors.Wrapf(err, "IsGuestHasDisk")
	}
	if !has {
		return nil
	}
	if err := c.Detach(diskId, guestId); err != nil {
		return errors.Wrapf(err, "Detach disk %s", diskId)
	}
	return c.WaitDiskDetach(diskId, guestId)
}

func (c *DisksHelper) Detach(diskId string, guestId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(diskId), "disk_id")
	s := c.session
	_, err := modules.Servers.PerformAction(s, guestId, "detachdisk", params)
	if err != nil {
		return err
	}
	return nil
}

func (c *DisksHelper) IsGuestHasDisk(diskId string, guestId string) (bool, error) {
	_, err := c.GetIndexOfGuest(diskId, guestId)
	if err != nil {
		if err1 := errors.Cause(err); err1 == GuestNoDiskError {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *DisksHelper) GetHostStorages(hostId string) ([]types.SHostStorage, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "details")
	ret, err := modules.Hoststorages.ListDescendent(c.session, hostId, params)
	if err != nil {
		return nil, errors.Wrapf(err, "Get host %s attached storages", hostId)
	}
	hostStorages := []types.SHostStorage{}
	for _, obj := range ret.Data {
		s := types.SHostStorage{}
		if err := obj.Unmarshal(&s); err != nil {
			return nil, errors.Wrapf(err, "Unmarshal host storage: %s", obj.String())
		}
		hostStorages = append(hostStorages, s)
	}
	return hostStorages, nil
}

func (c *DisksHelper) IsHostHasDisk(diskId string, hostId string) (bool, error) {
	hostStorages, err := c.GetHostStorages(hostId)
	if err != nil {
		return false, errors.Wrapf(err, "GetHostStorages")
	}
	for _, s := range hostStorages {
		if diskId == s.StorageId {
			return true, nil
		}
	}
	return false, nil
}

func (c *DisksHelper) WaitDiskAttach(diskId string, guestId string) error {
	doneF := func() (bool, error) {
		has, err := c.IsGuestHasDisk(diskId, guestId)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
		log.Warningf("Disk %s not attach from %s guest yet", diskId, guestId)
		return false, nil
	}
	return c.Wait(doneF)
}

func (c *DisksHelper) WaitDiskDetach(diskId string, guestId string) error {
	doneF := func() (bool, error) {
		has, err := c.IsGuestHasDisk(diskId, guestId)
		if err != nil {
			return false, err
		}
		if !has {
			return true, nil
		}
		log.Warningf("Disk %s not detach from %s guest yet", diskId, guestId)
		return false, nil
	}
	return c.Wait(doneF)
}

func (c *DisksHelper) EnsureDelete(diskId string) error {
	if err := c.DeleteDisk(diskId); err != nil {
		return errors.Wrapf(err, "Delete disk %s", diskId)
	}
	return c.WaitObjectDelete(diskId, false, continueWaitResource)
}

var (
	GuestNoDiskError = errors.New("Guest not has disk")
)

func (c *DisksHelper) GetIndexOfGuest(diskId string, guestId string) (int, error) {
	s := c.session
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "admin")
	params.Add(jsonutils.JSONTrue, "system")
	ret, err := modules.Serverdisks.ListDescendent(s, guestId, params)
	if err != nil {
		return 0, errors.Errorf("List guest %s disks: %v", guestId, err)
	}
	items := ret.Data
	if len(items) == 0 {
		return 0, errors.Wrapf(GuestNoDiskError, "Disk %s not attach to guest %s", diskId, guestId)
	}
	for _, item := range items {
		id, err := item.GetString("disk_id")
		if err != nil {
			return 0, errors.Errorf("Disk %s not has disk_id", item.String())
		}
		if id == diskId {
			index, err := item.Int("index")
			if err != nil {
				return 0, errors.Errorf("Disk %s not has index", item.String())
			}
			return int(index), nil
		}
	}
	return 0, errors.Wrapf(GuestNoDiskError, "Not found disk %s in guest %s", diskId, guestId)
}

type ServersHelper struct {
	*ResourceHelper
}

func NewServersHelper(s *mcclient.ClientSession) *ServersHelper {
	return &ServersHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Servers),
	}
}

func (s *ServersHelper) Get(id string) (*types.SGuest, error) {
	obj, err := s.ResourceHelper.Get(s.session, id, nil)
	if err != nil {
		return nil, errors.Errorf("Get vm by name %q: %v", id, err)
	}
	srv := new(types.SGuest)
	err = obj.Unmarshal(srv)
	return srv, err
}

type HostsHelper struct {
	*ResourceHelper
}

func NewHostsHelper(s *mcclient.ClientSession) *HostsHelper {
	return &HostsHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Hosts),
	}
}

func (s *HostsHelper) Get(id string) (*types.SHost, error) {
	obj, err := s.ResourceHelper.Get(s.session, id, nil)
	if err != nil {
		return nil, errors.Errorf("Get host by name %q: %v", id, err)
	}
	host := new(types.SHost)
	err = obj.Unmarshal(host)
	return host, err
}

const (
	NotFoundMsg = "NotFoundError"
)

func IsNotFoundError(err error) bool {
	if httpErr, ok := err.(*httputils.JSONClientError); ok {
		if httpErr.Code == 404 {
			return true
		}
	}
	if strings.Contains(err.Error(), NotFoundMsg) {
		return true
	}
	return false
}
