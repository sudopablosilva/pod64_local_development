    function run() {
ENDPOINT="localhost:8080"

#ENDPOINT="job-manager-worker-service.job-manager-worker:30447"
EXECUTION_UUID=$(curl -v --noproxy localhost -X POST http://$ENDPOINT/start \
-H "Content-Type: application/json" \
-d '
{
    "executionName": "NS7GL333",
    "accountId": "017820684888",
    "commonProperties": {
        "accountId": "017820684888",
        "providerConfigRef": "",
        "region": "sa-east-1",
        "tags": {
            "proprietario-equipe-e-mail": "douglas.pinheiro-santos@itau-unibanco.com.br",
            "tech-team-email": "ItauBatchOnCloud@itau-unibanco.com.br"
        }
    },
    "runtimes": [
        {
            "compute": {
                "type": "sampleruntime",
                "description": "Job de carregamento de works e temps"
            },
            "runtimeName": "jp7w",
            "security": {
                "cloudwatchEncryptionMode": "SSE-KMS",
                "executionRoleArn": "arn:aws:iam::017820684888:role/iamsr/role-glue-test-simples-iamsr",
                "jobBookmarksEncryptionMode": "CSE-KMS",
                "kmsKeyArn": "arn:aws:kms:sa-east-1:017820684888:key/b2263ba0-d5c2-490a-a7d6-0b7f4f4d0cb7",
                "s3EncryptionEncryptionMode": "SSE-KMS",
                "securityConfigurationName": "gluesecurityconfiguration"
            },
            "tags": {
                "valor1": "xpto"
            }
        },
        {
            "compute": {
                "type": "sampleruntime",
                "description": "Job de carregamento de works e temps"
            },
            "runtimeName": "jp7h",
            "security": {
                "cloudwatchEncryptionMode": "SSE-KMS",
                "executionRoleArn": "arn:aws:iam::017820684888:role/iamsr/role-glue-test-simples-iamsr",
                "jobBookmarksEncryptionMode": "CSE-KMS",
                "kmsKeyArn": "arn:aws:kms:sa-east-1:017820684888:key/b2263ba0-d5c2-490a-a7d6-0b7f4f4d0cb7",
                "s3EncryptionEncryptionMode": "SSE-KMS",
                "securityConfigurationName": "gluesecurityconfiguration"
            },
            "tags": {
                "valor1": "xpto"
            }
        },
        {
            "compute": {
                "type": "sampleruntime",
                "description": "Job de carregamento de works e temps"
            },
            "runtimeName": "jp7f",
            "security": {
                "cloudwatchEncryptionMode": "SSE-KMS",
                "executionRoleArn": "arn:aws:iam::017820684888:role/iamsr/role-glue-test-simples-iamsr",
                "jobBookmarksEncryptionMode": "CSE-KMS",
                "kmsKeyArn": "arn:aws:kms:sa-east-1:017820684888:key/b2263ba0-d5c2-490a-a7d6-0b7f4f4d0cb7",
                "s3EncryptionEncryptionMode": "SSE-KMS",
                "securityConfigurationName": "gluesecurityconfiguration"
            },
            "tags": {
                "valor1": "xpto"
            }
        },
        {
            "compute": {
                "type": "sampleruntime",
                "description": "Job de carregamento de works e temps"
            },
            "runtimeName": "jp7g",
            "security": {
                "cloudwatchEncryptionMode": "SSE-KMS",
                "executionRoleArn": "arn:aws:iam::017820684888:role/iamsr/role-glue-test-simples-iamsr",
                "jobBookmarksEncryptionMode": "CSE-KMS",
                "kmsKeyArn": "arn:aws:kms:sa-east-1:017820684888:key/b2263ba0-d5c2-490a-a7d6-0b7f4f4d0cb7",
                "s3EncryptionEncryptionMode": "SSE-KMS",
                "securityConfigurationName": "gluesecurityconfiguration"
            },
            "tags": {
                "valor1": "xpto"
            }
        }
    ],
    "schedulerRoutine": {
        "executionName": "JP799999",
        "cron": "0 6 * * *",
        "dependsOn": "None",
        "priority": "medium",
        "provisioning": "manual",
        "steps": [
            {
                "stepId": "JP799999-1",
                "tasks": [
                    {
                        "taskId": "JP7F001G",
                        "runtimeName": "jp7f",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobtempfinalclie",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "tmp_cliente",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7G002G",
                        "runtimeName": "jp7g",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobtempfinalctrt",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "tmp_contrato",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    }
                ]
            },
            {
                "stepId": "JP799999-2",
                "tasks": [
                    {
                        "taskId": "JP7H001G",
                        "runtimeName": "jp7h",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobgluehistoricalworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms001",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7H002G  xxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                        "runtimeName": "jp7h",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobgluehistoricalworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms002",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7H004G",
                        "runtimeName": "jp7h",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobgluehistoricalworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms004",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7H008G",
                        "runtimeName": "jp7h",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobgluehistoricalworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms008",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W103G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms103",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W009G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms109",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    }
                ]
            },
            {
                "stepId": "JP799999-3",
                "tasks": [
                    {
                        "taskId": "JP7W117G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms117",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W005G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms105",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W007G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms007",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W101G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms101",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W108G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms108",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W106G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms106",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    }
                ]
            },
            {
                "stepId": "JP799999-4",
                "tasks": [
                    {
                        "taskId": "JP7W018G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms018",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W002G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms002",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W011G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms011",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W015G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms015",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W016G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms016",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    },
                    {
                        "taskId": "JP7W017G",
                        "runtimeName": "jp7w",
                        "parameters": {
                            "PARM1": "estrategiasrecupcredito-jobglueworks",
                            "PARM2": "arn:aws:iam::076473262157:role/role-devops1-tradops",
                            "PARM3": "--pgms",
                            "PARM4": "pgms017",
                            "PARM5": "None",
                            "PARM6": "None",
                            "PARM7": "5"
                        }
                    }
                ]
            }
        ]
    }
}
')
}
for i in {1..1}; do (run); done