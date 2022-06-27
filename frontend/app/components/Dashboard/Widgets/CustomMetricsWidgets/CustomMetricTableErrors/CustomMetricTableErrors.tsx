import React, { useEffect } from 'react';
import { Pagination, NoContent } from 'UI';
import ErrorListItem from '../../../components/Errors/ErrorListItem';
import { withRouter, RouteComponentProps } from 'react-router-dom';
import { useModal } from 'App/components/Modal';
import ErrorDetailsModal from '../../../components/Errors/ErrorDetailsModal';

const PER_PAGE = 5;
interface Props {
    metric: any;
    isTemplate?: boolean;
    isEdit?: boolean;
    history: any,
    location: any,
}
function CustomMetricTableErrors(props: RouteComponentProps<Props>) {
    const { metric, isEdit = false } = props;
    const errorId = new URLSearchParams(props.location.search).get("errorId");
    const { showModal, hideModal } = useModal();

    const onErrorClick = (e: any, error: any) => {
        e.stopPropagation();
        props.history.replace({search: (new URLSearchParams({errorId : error.errorId})).toString()});
    }

    useEffect(() => {
        if (!errorId) return;

        showModal(<ErrorDetailsModal errorId={errorId} />, { right: true, onClose: () => {
            if (props.history.location.pathname.includes("/dashboard")) {
                props.history.replace({search: ""});
            }
        }});

        return () => {
            hideModal();
        }
    }, [errorId])

    return (
        <NoContent
            show={!metric.data.errors || metric.data.errors.length === 0}
            size="small"
        >
            <div className="pb-4">
                {metric.data.errors && metric.data.errors.map((error: any, index: any) => (
                    <div key={index} className="broder-b last:border-none">
                        <ErrorListItem error={error} onClick={(e) => onErrorClick(e, error)} />
                    </div>
                ))}

                {isEdit && (
                    <div className="my-6 flex items-center justify-center">
                        <Pagination
                            page={metric.page}
                            totalPages={Math.ceil(metric.data.total / metric.limit)}
                            onPageChange={(page: any) => metric.updateKey('page', page)}
                            limit={metric.limit}
                            debounceRequest={500}
                        />
                    </div>
                )}

                {!isEdit && (
                    <ViewMore total={metric.data.total} limit={metric.limit} />
                )}
            </div>
        </NoContent>
    );
}

export default withRouter(CustomMetricTableErrors) as React.FunctionComponent<RouteComponentProps<Props>>;

const ViewMore = ({ total, limit }: any) => total > limit && (
    <div className="mt-4 flex items-center justify-center cursor-pointer w-fit mx-auto">
        <div className="text-center">
            <div className="color-teal text-lg">
                All <span className="font-medium">{total}</span> errors
            </div>
        </div>
    </div>
);