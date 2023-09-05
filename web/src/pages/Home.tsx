'use client'

import { ReactNode } from 'react'
import {
    Box,
    Flex,
    useColorModeValue,
    chakra,
    SimpleGrid,
    Stat,
    StatLabel,
    StatNumber,
    Card,
    CardHeader,
    CardBody,
    Text,
} from '@chakra-ui/react'
import {
    FiServer,
} from 'react-icons/fi'
import { BsPerson } from 'react-icons/bs'
import { GoLocation } from 'react-icons/go'


interface StatsCardProps {
    title: string
    stat: string
    icon: ReactNode
}

function StatsCard(props: StatsCardProps) {
    const { title, stat, icon } = props
    return (
        <Stat
            px={{ base: 2, md: 4 }}
            py={'5'}
            shadow={'xl'}
            border={'1px solid'}
            borderColor={useColorModeValue('gray.800', 'gray.500')}
            rounded={'lg'}>
            <Flex justifyContent={'space-between'}>
                <Box pl={{ base: 2, md: 4 }}>
                    <StatLabel fontWeight={'medium'} isTruncated>
                        {title}
                    </StatLabel>
                    <StatNumber fontSize={'2xl'} fontWeight={'medium'}>
                        {stat}
                    </StatNumber>
                </Box>
                <Box
                    my={'auto'}
                    color={useColorModeValue('gray.800', 'gray.200')}
                    alignContent={'center'}>
                    {icon}
                </Box>
            </Flex>
        </Stat>
    )
}

export default function Home() {
    const textColor = useColorModeValue("gray.700", "white");
    return (
        <Box maxW="7xl" mx={'auto'} pt={5} px={{ base: 2, sm: 12, md: 17 }}>
            <Card my='22px' overflowX={{ sm: "scroll", xl: "hidden" }}>
                <CardHeader>
                    <Flex direction='column'>
                        <Text fontSize='lg' color={textColor} fontWeight='bold' pb='.5rem'>
                            Server status
                        </Text>
                    </Flex>
                </CardHeader>
                <CardBody>

                    <chakra.h1 textAlign={'center'} fontSize={'4xl'} py={10} fontWeight={'bold'}>
                        Our company is expanding, you could be too.
                    </chakra.h1>
                    <SimpleGrid columns={{ base: 1, md: 3 }} spacing={{ base: 5, lg: 8 }}>
                        <StatsCard title={'Users'} stat={'5,000'} icon={<BsPerson size={'3em'} />} />
                        <StatsCard title={'Servers'} stat={'1,000'} icon={<FiServer size={'3em'} />} />
                        <StatsCard title={'Datacenters'} stat={'7'} icon={<GoLocation size={'3em'} />} />
                    </SimpleGrid>
                </CardBody>
            </Card>
        </Box >
    )
}