import { DecoratorFn, Meta, Story } from '@storybook/react'
import * as H from 'history'
import { MATCH_ANY_PARAMETERS, WildcardMockLink } from 'wildcard-mock-link'

import { getDocumentNode } from '@sourcegraph/http-client'
import { ExternalServiceKind } from '@sourcegraph/shared/src/graphql-operations'
import { NOOP_TELEMETRY_SERVICE } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { MockedTestProvider } from '@sourcegraph/shared/src/testing/apollo'

import { WebStory } from '../components/WebStory'
import { WebhookFields } from '../graphql-operations'

import { WEBHOOKS } from './backend'
import { SiteAdminWebhooksPage } from './SiteAdminWebhooksPage'

const decorator: DecoratorFn = Story => <Story />

const config: Meta = {
    title: 'web/src/site-admin/SiteAdminWebhooksPage',
    decorators: [decorator],
}

export default config

export const NoWebhooksFound: Story = () => (
    <WebStory>
        {() => (
            <MockedTestProvider
                link={
                    new WildcardMockLink([
                        {
                            request: {
                                query: getDocumentNode(WEBHOOKS),
                                variables: MATCH_ANY_PARAMETERS,
                            },
                            result: {
                                data: {
                                    webhooks: {
                                        nodes: [],
                                        totalCount: 0,
                                        pageInfo: {
                                            hasNextPage: false,
                                        },
                                    },
                                },
                            },
                        },
                    ])
                }
            >
                <SiteAdminWebhooksPage
                    match={{} as any}
                    history={H.createMemoryHistory()}
                    location={{} as any}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                />
            </MockedTestProvider>
        )}
    </WebStory>
)

NoWebhooksFound.storyName = 'No webhooks found'

export const FiveWebhooksFound: Story = () => (
    <WebStory>
        {() => (
            <MockedTestProvider
                link={
                    new WildcardMockLink([
                        {
                            request: {
                                query: getDocumentNode(WEBHOOKS),
                                variables: MATCH_ANY_PARAMETERS,
                            },
                            result: {
                                data: {
                                    webhooks: {
                                        nodes: [
                                            createWebhookMock(
                                                'Bitbucket Cloud commit webhook',
                                                ExternalServiceKind.BITBUCKETCLOUD,
                                                'https://bitbucket.com/'
                                            ),
                                            createWebhookMock(
                                                'Github.com commit webhook',
                                                ExternalServiceKind.GITHUB,
                                                'https://github.com/'
                                            ),
                                            createWebhookMock(
                                                'Github.com PR push webhook',
                                                ExternalServiceKind.GITHUB,
                                                'https://github.com/'
                                            ),
                                            createWebhookMock(
                                                'Github.com PR creation webhook',
                                                ExternalServiceKind.GITHUB,
                                                'https://github.com/'
                                            ),
                                            createWebhookMock(
                                                'Bitbucket Cloud PR webhook',
                                                ExternalServiceKind.BITBUCKETCLOUD,
                                                'https://bitbucket.com/'
                                            ),
                                        ],
                                        totalCount: 5,
                                        pageInfo: {
                                            hasNextPage: false,
                                        },
                                    },
                                },
                            },
                        },
                    ])
                }
            >
                <SiteAdminWebhooksPage
                    match={{} as any}
                    history={H.createMemoryHistory()}
                    location={{} as any}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                />
            </MockedTestProvider>
        )}
    </WebStory>
)

FiveWebhooksFound.storyName = '5 webhooks found'

function createWebhookMock(name: string, kind: ExternalServiceKind, urn: string): WebhookFields {
    return {
        __typename: 'Webhook',
        createdAt: '',
        id: `webhook-${urn}`,
        name,
        secret: null,
        updatedAt: '',
        url: '',
        uuid: '',
        codeHostKind: kind,
        codeHostURN: urn,
        createdBy: {
            username: 'alice',
            url: 'users/alice',
        },
        updatedBy: {
            username: 'alice',
            url: 'users/alice',
        },
    }
}
