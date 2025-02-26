import Avatar from 'hew/Avatar';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import LogoGenAI from 'assets/images/logo-genai.svg?url';
import ActionSheet, { ActionItem } from 'components/ActionSheet';
import Link, { Props as LinkProps } from 'components/Link';
import useUI from 'components/ThemeProvider';
import UserSettings from 'components/UserSettings';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { handlePath, paths, serverAddress } from 'routes/utils';
import authStore from 'stores/auth';
import clusterStore from 'stores/cluster';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import { useObservable } from 'utils/observable';
import { AnyMouseEvent, routeToReactUrl } from 'utils/routes';

import css from './NavigationTabbar.module.scss';
import UserBadge from './UserBadge';
import WorkspaceCreateModalComponent from './WorkspaceCreateModal';

interface ToolbarItemProps extends LinkProps {
  badge?: number;
  icon: IconName;
  label: string;
  status?: string;
}

const ToolbarItem: React.FC<ToolbarItemProps> = ({ path, status, ...props }: ToolbarItemProps) => {
  const location = useLocation();
  const classes = [css.toolbarItem];
  const [isActive, setIsActive] = useState(false);

  if (isActive) classes.push(css.active);

  useEffect(() => setIsActive(location.pathname === path), [location.pathname, path]);

  return (
    <Link className={classes.join(' ')} path={path} {...props}>
      <Icon name={props.icon} size="large" title={props.label} />
      {status && <div className={css.status}>{status}</div>}
    </Link>
  );
};

const NavigationTabbar: React.FC = () => {
  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));

  const clusterStatus = useObservable(clusterStore.clusterStatus);

  const info = useObservable(determinedStore.info);
  const loadablePinnedWorkspaces = useObservable(workspaceStore.pinned);
  const pinnedWorkspaces = Loadable.getOrElse([], loadablePinnedWorkspaces);

  const { ui } = useUI();

  const [isShowingOverflow, setIsShowingOverflow] = useState(false);
  const [isShowingPinnedWorkspaces, setIsShowingPinnedWorkspaces] = useState(false);

  const showNavigation = isAuthenticated && ui.showChrome;

  const { canCreateWorkspace, canAdministrateUsers } = usePermissions();
  const gasLinkOn = useFeature().isOn('genai');

  const WorkspaceCreateModal = useModal(WorkspaceCreateModalComponent);

  const [showSettings, setShowSettings] = useState<boolean>(false);

  const handleOverflowOpen = useCallback(() => setIsShowingOverflow(true), []);
  const handleWorkspacesOpen = useCallback(() => {
    if (pinnedWorkspaces.length === 0) {
      routeToReactUrl(paths.workspaceList());
      return;
    }
    setIsShowingPinnedWorkspaces(true);
  }, [pinnedWorkspaces]);
  const handleActionSheetCancel = useCallback(() => {
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  const handlePathUpdate = useCallback((e: AnyMouseEvent, path?: string) => {
    handlePath(e, { path });
    setIsShowingOverflow(false);
    setIsShowingPinnedWorkspaces(false);
  }, []);

  if (!showNavigation) return null;

  const workspaceActions = Loadable.quickMatch(
    loadablePinnedWorkspaces,
    [{ icon: <Spinner spinning />, label: 'Loading...' }],
    [{ icon: <Spinner spinning />, label: 'Could not load workspaces' }], // TODO use proper icon here
    (workspaces) =>
      workspaces.map(
        (workspace) =>
          ({
            icon: <Avatar palette="muted" square text={workspace.name} />,
            label: workspace.name,
            onClick: (e: AnyMouseEvent) =>
              handlePathUpdate(e, paths.workspaceDetails(workspace.id)),
          }) as ActionItem,
      ),
  );

  if (canCreateWorkspace) {
    workspaceActions.push({
      icon: (
        <div className={css.newWorkspaceIcon}>
          <Icon decorative name="add" size="tiny" />
        </div>
      ),
      label: 'New Workspace',
      onClick: WorkspaceCreateModal.open,
    });
  }

  interface OverflowActionProps {
    external?: boolean;
    icon?: IconName | JSX.Element;
    label: string;
    onClick?: (e: AnyMouseEvent) => void;
    path?: string;
    popout?: boolean;
    render?: () => JSX.Element;
  }

  const overflowActionsTop: OverflowActionProps[] = [
    {
      label: 'Current user',
      render: () => (
        <div className={css.user}>
          <UserBadge compact key="avatar" user={currentUser} />
        </div>
      ),
    },
  ];

  if (canAdministrateUsers) {
    overflowActionsTop.push({
      icon: 'group',
      label: 'Admin Settings',
      onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.admin()),
    });
  }

  const overflowActionsBottom: OverflowActionProps[] = [
    {
      icon: 'settings',
      label: 'User Settings',
      onClick: () => setShowSettings(true),
    },
    {
      icon: 'user',
      label: 'Sign out',
      onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.logout() + '?hard_logout=true'),
    },
    {
      external: true,
      icon: 'docs',
      label: 'Docs',
      path: paths.docs(),
      popout: true,
    },
    {
      external: true,
      icon: 'cloud',
      label: 'API (Beta)',
      path: paths.docs('/rest-api/'),
      popout: true,
    },
    {
      external: true,
      icon: 'pencil',
      label: 'Feedback',
      path: paths.submitProductFeedback(info.branding || BrandingType.Determined),
      popout: true,
    },
  ];

  if (gasLinkOn) {
    overflowActionsBottom.push({
      external: true,
      icon: <img alt="GenAI Studio" height={24} src={LogoGenAI} width={24} />,
      label: 'GenAI',
      path: serverAddress('/genai'),
      popout: true,
    });
  }

  return (
    <>
      <nav className={css.base}>
        <div className={css.toolbar}>
          <ToolbarItem icon="home" label="Home" path={paths.dashboard()} />
          <ToolbarItem icon="experiment" label="Uncategorized" path={paths.uncategorized()} />
          <ToolbarItem icon="model" label="Model Registry" path={paths.modelList()} />
          <ToolbarItem icon="tasks" label="Tasks" path={paths.taskList()} />
          <ToolbarItem
            icon="cluster"
            label="Cluster"
            path={paths.clusters()}
            status={clusterStatus}
          />
          <ToolbarItem icon="workspaces" label="Workspaces" onClick={handleWorkspacesOpen} />
          <ToolbarItem
            icon="overflow-vertical"
            label="Overflow Menu"
            onClick={handleOverflowOpen}
          />
        </div>
        <ActionSheet
          actions={[
            {
              icon: 'workspaces',
              label: 'Workspaces',
              onClick: (e: AnyMouseEvent) => handlePathUpdate(e, paths.workspaceList()),
              path: paths.workspaceList(),
            },
            ...workspaceActions,
          ]}
          show={isShowingPinnedWorkspaces}
          onCancel={handleActionSheetCancel}
        />
        <ActionSheet
          actions={[...overflowActionsTop, ...overflowActionsBottom]}
          show={isShowingOverflow}
          onCancel={handleActionSheetCancel}
        />
        <WorkspaceCreateModal.Component />
      </nav>
      <UserSettings show={showSettings} onClose={() => setShowSettings(false)} />
    </>
  );
};

export default NavigationTabbar;
