//@ts-nocheck
import React from 'react';
import { Icon } from 'UI';
import { checkForRecent } from 'App/date';
import { Tooltip } from 'react-tippy';

interface Props {
    user: any;
    editHandler?: any;
    generateInvite?: any;
    copyInviteCode?: any;
}
function UserListItem(props: Props) {
    const {
        user,
        editHandler = () => {},
        generateInvite = () => {},
        copyInviteCode = () => {},
    } = props;
    return (
        <div className="grid grid-cols-12 p-3 py-4 border-b items-center select-none hover:bg-active-blue group">
            <div className="col-span-5">
                {user.name}
                {user.isAdmin && <span className="ml-2 px-2 py-1 bg-gray-lightest rounded border text-sm capitalize">Admin</span>}
                {user.isSuperAdmin && <span className="ml-2 px-2 py-1 bg-gray-lightest rounded border text-sm capitalize">Owner</span>}
            </div>
            <div className="col-span-3">
                <span className="px-2 py-1 bg-gray-lightest rounded border text-sm capitalize">
                    {user.roleName}
                </span>
            </div>
            <div className="col-span-2">
                <span>{user.createdAt && checkForRecent(user.createdAt, 'LLL dd, yyyy, hh:mm a')}</span>
            </div>

            <div className="col-span-2 justify-self-end invisible group-hover:visible">
                <div className="grid grid-cols-2 gap-3 items-center justify-end">
                    {!user.isJoined && user.invitationLink ? (
                        <Tooltip
                            delay={500}
                            arrow
                            title="Copy Invite Code"
                            hideOnClick={true}
                        >
                            <button className='' onClick={copyInviteCode}>
                                <Icon name="link-45deg" size="16" color="teal"/>
                            </button>
                        </Tooltip>
                    ) : <div/>}
                    {!user.isJoined && user.isExpiredInvite && (
                        <Tooltip
                            delay={500}
                            arrow
                            title="Generate Invite"
                            hideOnClick={true}
                        >
                            <button className='' onClick={generateInvite}>
                                <Icon name="link-45deg" size="16" color="red"/>
                            </button>
                        </Tooltip>
                    )}
                    <button className='' onClick={editHandler}>
                        <Icon name="pencil" color="teal" size="16" />
                    </button>
                </div>
            </div>
        </div>
    );
}

export default UserListItem;