import { deletableRunStates, killableRunStates, terminalRunStates } from 'constants/states';
import { PermissionsHook } from 'hooks/usePermissions';
import { FlatRun, FlatRunAction } from 'types';

type FlatRunChecker = (flatRun: FlatRun) => boolean;

type FlatRunPermissionSet = Pick<
  PermissionsHook,
  'canCreateExperiment' | 'canDeleteFlatRun' | 'canModifyExperiment' | 'canMoveFlatRun'
>;

const flatRunCheckers: Record<FlatRunAction, FlatRunChecker> = {
  [FlatRunAction.Archive]: (flatRun) =>
    !flatRun.archived && !flatRun.archived && terminalRunStates.has(flatRun.state),

  [FlatRunAction.Delete]: (flatRun) => deletableRunStates.has(flatRun.state),

  [FlatRunAction.Kill]: (flatRun) => killableRunStates.includes(flatRun.state),

  [FlatRunAction.Move]: (flatRun) => !flatRun.archived && !flatRun.archived,

  // [FlatRunAction.Pause]: (run) => pausableRunStates.has(run.state),

  [FlatRunAction.Unarchive]: (flatRun) => terminalRunStates.has(flatRun.state) && flatRun.archived,
};

export const canActionFlatRun = (action: FlatRunAction, flatRun: FlatRun): boolean => {
  return flatRunCheckers[action](flatRun);
};

const getActionsForFlatRun = (
  flatRun: FlatRun,
  targets: FlatRunAction[],
  permissions: FlatRunPermissionSet,
): FlatRunAction[] => {
  if (!flatRun) return []; // redundant, for clarity
  const workspace = { id: flatRun.workspaceId };
  return targets
    .filter((action) => canActionFlatRun(action, flatRun))
    .filter((action) => {
      switch (action) {
        case FlatRunAction.Delete:
          return permissions.canDeleteFlatRun({ flatRun });

        case FlatRunAction.Move:
          return permissions.canMoveFlatRun({ flatRun });

        case FlatRunAction.Archive:
        case FlatRunAction.Unarchive:
          return permissions.canModifyExperiment({ workspace });

        case FlatRunAction.Kill:
          return permissions.canModifyExperiment({ workspace }) && !flatRun.experiment?.unmanaged;

        default:
          return true;
      }
    });
};

export const getActionsForFlatRunsUnion = (
  flatRun: FlatRun[],
  targets: FlatRunAction[],
  permissions: FlatRunPermissionSet,
): FlatRunAction[] => {
  if (!flatRun.length) return [];
  const actionsForRuns = flatRun.map((run) => getActionsForFlatRun(run, targets, permissions));
  return targets.filter((action) =>
    actionsForRuns.some((runActions) => runActions.includes(action)),
  );
};