import React, { useCallback, useState } from 'react'

import { ErrorAlert } from '@sourcegraph/branded/src/components/alerts'
import { Form } from '@sourcegraph/branded/src/components/Form'
import { logger } from '@sourcegraph/common'
import { Button, Modal, Input, H3, Text } from '@sourcegraph/wildcard'

import { LoaderButton } from '../../../components/LoaderButton'
import { ExecutorSecretScope, Scalars } from '../../../graphql-operations'

import { useCreateExecutorSecret } from './backend'

export interface AddSecretModalProps {
    onCancel: () => void
    afterCreate: () => void
    namespaceID: Scalars['ID'] | null
    scope: ExecutorSecretScope
}

export const AddSecretModal: React.FunctionComponent<React.PropsWithChildren<AddSecretModalProps>> = ({
    onCancel,
    afterCreate,
    namespaceID,
    scope,
}) => {
    const labelId = 'addSecret'

    const [key, setKey] = useState<string>('')
    const onChangeKey = useCallback<React.ChangeEventHandler<HTMLInputElement>>(event => {
        setKey(event.target.value)
    }, [])

    const [value, setValue] = useState<string>('')
    const onChangeValue = useCallback<React.ChangeEventHandler<HTMLInputElement>>(event => {
        setValue(event.target.value)
    }, [])

    const [createExecutorSecret, { loading, error }] = useCreateExecutorSecret()

    const onSubmit = useCallback<React.FormEventHandler>(
        async event => {
            event.preventDefault()

            try {
                await createExecutorSecret({
                    variables: {
                        namespace: namespaceID,
                        key,
                        value,
                        scope,
                    },
                })

                afterCreate()
            } catch (error) {
                // Non-request error. API errors will be available under `error` above.
                logger.error(error)
            }
        },
        [createExecutorSecret, namespaceID, key, value, scope, afterCreate]
    )

    return (
        <Modal onDismiss={onCancel} aria-labelledby={labelId}>
            <H3 id={labelId}>Add new executor secret</H3>
            <Text>
                Executor secrets are available to executor jobs as environment variables. They will never appear in
                logs.
            </Text>
            {error && <ErrorAlert error={error} />}
            <Form onSubmit={onSubmit}>
                <div className="form-group">
                    <Input
                        id="key"
                        name="key"
                        autoComplete="off"
                        inputClassName="mb-2"
                        className="mb-0"
                        required={true}
                        spellCheck="false"
                        minLength={1}
                        value={key}
                        onChange={onChangeKey}
                        pattern="^[A-Z][A-Z0-9_]*$"
                        message={
                            <>
                                Must be uppercase characters, digits and underscores only. Must start with an uppercase
                                character.
                            </>
                        }
                        label="Key"
                    />
                </div>
                <div className="form-group">
                    <Input
                        id="value"
                        name="value"
                        type="password"
                        autoComplete="off"
                        required={true}
                        spellCheck="false"
                        minLength={1}
                        label="Value"
                        value={value}
                        onChange={onChangeValue}
                    />
                </div>
                <div className="d-flex justify-content-end">
                    <Button disabled={loading} className="mr-2" onClick={onCancel} outline={true} variant="secondary">
                        Cancel
                    </Button>
                    <LoaderButton
                        type="submit"
                        disabled={loading || key.length === 0 || value.length === 0}
                        variant="primary"
                        loading={loading}
                        alwaysShowLabel={true}
                        label="Add secret"
                    />
                </div>
            </Form>
        </Modal>
    )
}
